package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stripe/stripe-go/v78"
)

func TestMapStripeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected models.SubscriptionStatus
	}{
		{"active", models.StatusActive},
		{"past_due", models.StatusPastDue},
		{"canceled", models.StatusCanceled},
		{"paused", models.StatusPaused},
		{"trialing", models.StatusTrialing},
		{"unknown", models.StatusCanceled},
		{"", models.StatusCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapStripeStatus(tt.input)
			if got != tt.expected {
				t.Errorf("mapStripeStatus(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripeWebhook_DevMode_InvalidJSON(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStripeWebhook_ProductionMode_InvalidSignature(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "whsec_real_secret_123")

	payload := `{"id":"evt_123","type":"customer.subscription.created"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
	req.Header.Set("Stripe-Signature", "t=123,v1=invalid_signature")
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStripeWebhook_SubscriptionCreatedUpdated_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	subPayload := stripe.Subscription{
		ID: "sub_12345",
		Customer: &stripe.Customer{
			ID: "cus_99999",
		},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: 1700000000,
		CurrentPeriodEnd:   1702500000,
	}
	subRaw, _ := json.Marshal(subPayload)

	event := stripe.Event{
		ID:   "evt_001",
		Type: "customer.subscription.created",
		Data: &stripe.EventData{
			Raw: subRaw,
		},
	}
	eventBytes, _ := json.Marshal(event)

	mockDB.ExpectExec(`UPDATE subscriptions`).
		WithArgs(models.StatusActive, "sub_12345", pgxmock.AnyArg(), pgxmock.AnyArg(), "cus_99999").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(eventBytes))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled db expectations: %v", err)
	}
}

func TestStripeWebhook_SubscriptionCreated_DBError(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	subPayload := stripe.Subscription{
		ID: "sub_12345",
		Customer: &stripe.Customer{
			ID: "cus_99999",
		},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: 1700000000,
		CurrentPeriodEnd:   1702500000,
	}
	subRaw, _ := json.Marshal(subPayload)

	event := stripe.Event{
		ID:   "evt_001",
		Type: "customer.subscription.created",
		Data: &stripe.EventData{
			Raw: subRaw,
		},
	}
	eventBytes, _ := json.Marshal(event)

	mockDB.ExpectExec(`UPDATE subscriptions`).
		WithArgs(models.StatusActive, "sub_12345", pgxmock.AnyArg(), pgxmock.AnyArg(), "cus_99999").
		WillReturnError(errors.New("db connection timeout"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(eventBytes))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled db expectations: %v", err)
	}
}

func TestStripeWebhook_SubscriptionDeleted_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	subPayload := stripe.Subscription{
		ID: "sub_12345",
		Customer: &stripe.Customer{
			ID: "cus_99999",
		},
		Status: stripe.SubscriptionStatusCanceled,
	}
	subRaw, _ := json.Marshal(subPayload)

	event := stripe.Event{
		ID:   "evt_002",
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{
			Raw: subRaw,
		},
	}
	eventBytes, _ := json.Marshal(event)

	mockDB.ExpectExec(`UPDATE subscriptions`).
		WithArgs(models.StatusCanceled, pgxmock.AnyArg(), "cus_99999").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(eventBytes))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled db expectations: %v", err)
	}
}

func TestStripeWebhook_InvoicePaymentFailed_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	invPayload := stripe.Invoice{
		ID: "in_123",
		Customer: &stripe.Customer{
			ID: "cus_99999",
		},
	}
	invRaw, _ := json.Marshal(invPayload)

	event := stripe.Event{
		ID:   "evt_003",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{
			Raw: invRaw,
		},
	}
	eventBytes, _ := json.Marshal(event)

	mockDB.ExpectExec(`UPDATE subscriptions`).
		WithArgs(models.StatusPastDue, "cus_99999").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(eventBytes))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled db expectations: %v", err)
	}
}

func TestStripeWebhook_InvoicePaymentSucceeded_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	invPayload := stripe.Invoice{
		ID: "in_124",
		Customer: &stripe.Customer{
			ID: "cus_99999",
		},
	}
	invRaw, _ := json.Marshal(invPayload)

	event := stripe.Event{
		ID:   "evt_004",
		Type: "invoice.payment_succeeded",
		Data: &stripe.EventData{
			Raw: invRaw,
		},
	}
	eventBytes, _ := json.Marshal(event)

	mockDB.ExpectExec(`UPDATE subscriptions`).
		WithArgs(models.StatusActive, "cus_99999").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(eventBytes))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled db expectations: %v", err)
	}
}

func TestStripeWebhook_UnhandledEvent(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewSubscriptionHandler(mockDB, "")

	event := stripe.Event{
		ID:   "evt_999",
		Type: "customer.created",
	}
	eventBytes, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBuffer(eventBytes))
	w := httptest.NewRecorder()

	handler.StripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
