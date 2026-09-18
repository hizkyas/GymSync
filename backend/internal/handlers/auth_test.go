package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/middleware"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/pashagolub/pgxmock/v3"
	"golang.org/x/crypto/bcrypt"
)

const (
	testJWTSecret     = "super-secret-test-key-32-chars-long!"
	testJWTExpiryHours = 24
)

func TestRegister_InvalidBody(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRegister_ValidationErrors(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	tests := []struct {
		name string
		body models.RegisterRequest
	}{
		{
			name: "missing email",
			body: models.RegisterRequest{Email: "", Password: "password123", FullName: "John Doe"},
		},
		{
			name: "missing password",
			body: models.RegisterRequest{Email: "john@example.com", Password: "", FullName: "John Doe"},
		},
		{
			name: "missing full_name",
			body: models.RegisterRequest{Email: "john@example.com", Password: "password123", FullName: ""},
		},
		{
			name: "short password",
			body: models.RegisterRequest{Email: "john@example.com", Password: "short", FullName: "John Doe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(b))
			w := httptest.NewRecorder()

			handler.Register(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestRegister_DuplicateEmailConflict(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	body := models.RegisterRequest{
		Email:    "existing@example.com",
		Password: "password123",
		FullName: "Existing User",
	}
	b, _ := json.Marshal(body)

	mockDB.ExpectQuery(`INSERT INTO users`).
		WithArgs("existing@example.com", pgxmock.AnyArg(), "Existing User", models.RoleMember, pgxmock.AnyArg()).
		WillReturnError(errors.New("duplicate key value violates unique constraint users_email_key"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestRegister_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	body := models.RegisterRequest{
		Email:    "  NEWUSER@example.com  ",
		Password: "securepassword123",
		FullName: "New Member",
	}
	b, _ := json.Marshal(body)

	userID := uuid.New()
	qrToken := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "email", "full_name", "role", "phone", "avatar_url", "qr_token", "is_active", "created_at", "updated_at",
	}).AddRow(
		userID, "newuser@example.com", "New Member", models.RoleMember, nil, nil, qrToken, true, now, now,
	)

	mockDB.ExpectQuery(`INSERT INTO users`).
		WithArgs("newuser@example.com", pgxmock.AnyArg(), "New Member", models.RoleMember, pgxmock.AnyArg()).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}
	if resp.User.Email != "newuser@example.com" {
		t.Errorf("expected email newuser@example.com, got %s", resp.User.Email)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled db expectations: %v", err)
	}
}

func TestLogin_ValidationErrors(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	// Missing fields
	body := models.LoginRequest{Email: "", Password: ""}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	body := models.LoginRequest{Email: "nonexistent@example.com", Password: "password123"}
	b, _ := json.Marshal(body)

	mockDB.ExpectQuery(`SELECT id, email, password_hash`).
		WithArgs("nonexistent@example.com").
		WillReturnError(errors.New("no rows in result set"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_DeactivatedUser(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	userID := uuid.New()
	qrToken := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "email", "password_hash", "full_name", "role", "phone", "avatar_url", "qr_token", "is_active", "created_at", "updated_at",
	}).AddRow(
		userID, "deactive@example.com", string(hash), "Deactive User", models.RoleMember, nil, nil, qrToken, false, now, now,
	)

	mockDB.ExpectQuery(`SELECT id, email, password_hash`).
		WithArgs("deactive@example.com").
		WillReturnRows(rows)

	body := models.LoginRequest{Email: "deactive@example.com", Password: "password123"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestLogin_IncorrectPassword(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	userID := uuid.New()
	qrToken := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "email", "password_hash", "full_name", "role", "phone", "avatar_url", "qr_token", "is_active", "created_at", "updated_at",
	}).AddRow(
		userID, "user@example.com", string(hash), "Test User", models.RoleMember, nil, nil, qrToken, true, now, now,
	)

	mockDB.ExpectQuery(`SELECT id, email, password_hash`).
		WithArgs("user@example.com").
		WillReturnRows(rows)

	body := models.LoginRequest{Email: "user@example.com", Password: "wrongpassword"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	userID := uuid.New()
	qrToken := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "email", "password_hash", "full_name", "role", "phone", "avatar_url", "qr_token", "is_active", "created_at", "updated_at",
	}).AddRow(
		userID, "user@example.com", string(hash), "Test User", models.RoleAdmin, nil, nil, qrToken, true, now, now,
	)

	mockDB.ExpectQuery(`SELECT id, email, password_hash`).
		WithArgs("user@example.com").
		WillReturnRows(rows)

	body := models.LoginRequest{Email: "user@example.com", Password: "correctpassword"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Role != models.RoleAdmin {
		t.Errorf("expected role admin, got %s", resp.User.Role)
	}
}

func TestMe_Unauthenticated(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	handler.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMe_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	handler := NewAuthHandler(mockDB, testJWTSecret, testJWTExpiryHours)

	userID := uuid.New()
	qrToken := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "email", "full_name", "role", "phone", "avatar_url", "qr_token", "is_active", "created_at", "updated_at",
	}).AddRow(
		userID, "trainer@example.com", "Trainer Joe", models.RoleTrainer, nil, nil, qrToken, true, now, now,
	)

	mockDB.ExpectQuery(`SELECT id, email, full_name`).
		WithArgs(userID).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID.String())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.Me(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to unmarshal user: %v", err)
	}

	if user.ID != userID {
		t.Errorf("expected user ID %s, got %s", userID, user.ID)
	}
}
