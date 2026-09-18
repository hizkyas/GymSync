package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the role of a user in the system.
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleTrainer UserRole = "trainer"
	RoleMember  UserRole = "member"
)

// User represents a gym member, trainer, or admin.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	FullName     string    `json:"full_name" db:"full_name"`
	Role         UserRole  `json:"role" db:"role"`
	Phone        *string   `json:"phone,omitempty" db:"phone"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	QRToken      uuid.UUID `json:"qr_token" db:"qr_token"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserWithSubscription is a join result used in the check-in flow.
type UserWithSubscription struct {
	User
	SubscriptionStatus *SubscriptionStatus `json:"subscription_status,omitempty" db:"subscription_status"`
}

// RegisterRequest is the payload for the register endpoint.
type RegisterRequest struct {
	Email    string   `json:"email"`
	Password string   `json:"password"`
	FullName string   `json:"full_name"`
	Role     UserRole `json:"role"`
	Phone    string   `json:"phone"`
}

// LoginRequest is the payload for the login endpoint.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is returned after successful login or registration.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Claims represents the JWT payload.
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Role   UserRole `json:"role"`
}
