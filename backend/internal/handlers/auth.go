package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/middleware"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthDB defines database operations required by AuthHandler.
type AuthDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// AuthHandler handles authentication routes.
type AuthHandler struct {
	db             AuthDB
	jwtSecret      string
	jwtExpiryHours int
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(db AuthDB, jwtSecret string, jwtExpiryHours int) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret, jwtExpiryHours: jwtExpiryHours}
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		respondError(w, http.StatusBadRequest, "email, password, and full_name are required")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Default role to member
	if req.Role == "" {
		req.Role = models.RoleMember
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	var phone *string
	if req.Phone != "" {
		phone = &req.Phone
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err = h.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role, phone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, full_name, role, phone, avatar_url, qr_token, is_active, created_at, updated_at
	`, req.Email, string(hash), req.FullName, req.Role, phone).Scan(
		&user.ID, &user.Email, &user.FullName, &user.Role,
		&user.Phone, &user.AvatarURL, &user.QRToken,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := middleware.GenerateToken(&user, h.jwtSecret, h.jwtExpiryHours)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err := h.db.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, role, phone, avatar_url, qr_token, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role,
		&user.Phone, &user.AvatarURL, &user.QRToken,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.IsActive {
		respondError(w, http.StatusForbidden, "account is deactivated")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := middleware.GenerateToken(&user, h.jwtSecret, h.jwtExpiryHours)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: user})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	err = h.db.QueryRow(ctx, `
		SELECT id, email, full_name, role, phone, avatar_url, qr_token, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, uid).Scan(
		&user.ID, &user.Email, &user.FullName, &user.Role,
		&user.Phone, &user.AvatarURL, &user.QRToken,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}
