package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

// SubscriptionHandler handles Stripe webhook events.
type SubscriptionHandler struct {
	db                  *pgxpool.Pool
	stripeWebhookSecret string
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(db *pgxpool.Pool, stripeWebhookSecret string) *SubscriptionHandler {
	return &SubscriptionHandler{db: db, stripeWebhookSecret: stripeWebhookSecret}
}

// StripeWebhook handles POST /api/v1/webhooks/stripe
func (h *SubscriptionHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the raw body for signature verification
	const maxBodyBytes = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	// If webhook secret is not configured, skip signature verification (dev mode)
	var event stripe.Event
	if h.stripeWebhookSecret != "" && h.stripeWebhookSecret != "whsec_placeholder" {
		sigHeader := r.Header.Get("Stripe-Signature")
		event, err = webhook.ConstructEvent(payload, sigHeader, h.stripeWebhookSecret)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid webhook signature")
			return
		}
	} else {
		// Development mode: parse without verification
		if err := json.Unmarshal(payload, &event); err != nil {
			respondError(w, http.StatusBadRequest, "invalid webhook payload")
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			respondError(w, http.StatusBadRequest, "failed to parse subscription event")
			return
		}
		h.upsertSubscription(ctx, w, &sub)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			respondError(w, http.StatusBadRequest, "failed to parse subscription deleted event")
			return
		}
		h.cancelSubscription(ctx, w, &sub)

	default:
		// Acknowledge unknown events
		w.WriteHeader(http.StatusOK)
	}
}

func (h *SubscriptionHandler) upsertSubscription(ctx context.Context, w http.ResponseWriter, sub *stripe.Subscription) {
	status := mapStripeStatus(string(sub.Status))
	customerID := sub.Customer.ID
	subscriptionID := sub.ID
	periodStart := time.Unix(sub.CurrentPeriodStart, 0).UTC()
	periodEnd := time.Unix(sub.CurrentPeriodEnd, 0).UTC()

	_, err := h.db.Exec(ctx, `
		UPDATE subscriptions
		SET
			status = $1,
			stripe_subscription_id = $2,
			current_period_start = $3,
			current_period_end = $4,
			updated_at = NOW()
		WHERE stripe_customer_id = $5
	`, status, subscriptionID, periodStart, periodEnd, customerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update subscription")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SubscriptionHandler) cancelSubscription(ctx context.Context, w http.ResponseWriter, sub *stripe.Subscription) {
	customerID := sub.Customer.ID
	now := time.Now().UTC()

	_, err := h.db.Exec(ctx, `
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = NOW()
		WHERE stripe_customer_id = $3
	`, models.StatusCanceled, now, customerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to cancel subscription")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// mapStripeStatus converts a Stripe subscription status string to our internal enum.
func mapStripeStatus(stripeStatus string) models.SubscriptionStatus {
	switch stripeStatus {
	case "active":
		return models.StatusActive
	case "past_due":
		return models.StatusPastDue
	case "canceled":
		return models.StatusCanceled
	case "paused":
		return models.StatusPaused
	case "trialing":
		return models.StatusTrialing
	default:
		return models.StatusCanceled
	}
}
