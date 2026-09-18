package models

import (
	"time"

	"github.com/google/uuid"
)

// PlanInterval represents the billing interval for a membership plan.
type PlanInterval string

const (
	IntervalMonthly PlanInterval = "monthly"
	IntervalYearly  PlanInterval = "yearly"
)

// MembershipPlan represents a gym membership tier.
type MembershipPlan struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	Name            string       `json:"name" db:"name"`
	Description     *string      `json:"description,omitempty" db:"description"`
	PriceCents      int          `json:"price_cents" db:"price_cents"`
	Currency        string       `json:"currency" db:"currency"`
	BillingInterval PlanInterval `json:"billing_interval" db:"billing_interval"`
	StripePriceID   *string      `json:"stripe_price_id,omitempty" db:"stripe_price_id"`
	IsActive        bool         `json:"is_active" db:"is_active"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at" db:"updated_at"`
}
