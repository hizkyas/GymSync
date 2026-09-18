package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus mirrors the PostgreSQL enum.
type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusPaused   SubscriptionStatus = "paused"
	StatusTrialing SubscriptionStatus = "trialing"
)

// Subscription represents a member's gym subscription.
type Subscription struct {
	ID                   uuid.UUID          `json:"id" db:"id"`
	UserID               uuid.UUID          `json:"user_id" db:"user_id"`
	PlanID               uuid.UUID          `json:"plan_id" db:"plan_id"`
	Status               SubscriptionStatus `json:"status" db:"status"`
	StripeCustomerID     *string            `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID *string            `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty" db:"current_period_start"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty" db:"current_period_end"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty" db:"canceled_at"`
	CreatedAt            time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at" db:"updated_at"`
}
