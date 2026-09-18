package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// CheckIn represents a single member access log entry.
type CheckIn struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	QRToken       uuid.UUID  `json:"qr_token" db:"qr_token"`
	AccessGranted bool       `json:"access_granted" db:"access_granted"`
	DenialReason  *string    `json:"denial_reason,omitempty" db:"denial_reason"`
	CheckedInAt   time.Time  `json:"checked_in_at" db:"checked_in_at"`
	IPAddress     *net.IP    `json:"ip_address,omitempty" db:"ip_address"`
	Notes         *string    `json:"notes,omitempty" db:"notes"`
}

// CheckInRequest is the payload from the front-desk scanner.
type CheckInRequest struct {
	QRToken string  `json:"qr_token"`
	Notes   *string `json:"notes,omitempty"`
}

// CheckInResponse is returned to the caller after processing a check-in.
type CheckInResponse struct {
	AccessGranted bool       `json:"access_granted"`
	DenialReason  *string    `json:"denial_reason,omitempty"`
	MemberName    string     `json:"member_name"`
	MemberEmail   string     `json:"member_email"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	CheckedInAt   time.Time  `json:"checked_in_at"`
	CheckInID     uuid.UUID  `json:"check_in_id"`
}

// RecentCheckIn is a lightweight struct for the dashboard feed.
type RecentCheckIn struct {
	CheckInID     uuid.UUID          `json:"check_in_id" db:"id"`
	MemberName    string             `json:"member_name" db:"full_name"`
	MemberEmail   string             `json:"member_email" db:"email"`
	AvatarURL     *string            `json:"avatar_url,omitempty" db:"avatar_url"`
	AccessGranted bool               `json:"access_granted" db:"access_granted"`
	DenialReason  *string            `json:"denial_reason,omitempty" db:"denial_reason"`
	Status        SubscriptionStatus `json:"subscription_status" db:"status"`
	CheckedInAt   time.Time          `json:"checked_in_at" db:"checked_in_at"`
}
