package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/models"
)

const (
	testSecret = "test-secret-key-32-chars-minimum!"
)

func TestGenerateToken_And_JWTAuthMiddleware(t *testing.T) {
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleAdmin,
	}

	tokenStr, err := GenerateToken(user, testSecret, 24)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		role := GetRole(r.Context())

		if userID != user.ID.String() {
			t.Errorf("expected userID %s, got %s", user.ID.String(), userID)
		}
		if role != models.RoleAdmin {
			t.Errorf("expected role admin, got %s", role)
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := JWTAuthMiddleware(testSecret)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestJWTAuthMiddleware_HeaderErrors(t *testing.T) {
	mw := JWTAuthMiddleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		authHeader string
	}{
		{"missing header", ""},
		{"malformed header format", "Token 12345"},
		{"invalid secret token", "Bearer " + func() string {
			u := &models.User{ID: uuid.New(), Email: "a@b.com", Role: models.RoleMember}
			tok, _ := GenerateToken(u, "wrong-secret-key-12345678901234567890", 24)
			return tok
		}()},
		{"expired token", "Bearer " + func() string {
			u := &models.User{ID: uuid.New(), Email: "a@b.com", Role: models.RoleMember}
			tok, _ := GenerateToken(u, testSecret, -1) // expired 1 hour ago
			return tok
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			mw.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	mw := RequireRole(models.RoleAdmin, models.RoleTrainer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: No role in context
	req1 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w1 := httptest.NewRecorder()
	mw.ServeHTTP(w1, req1)
	if w1.Code != http.StatusForbidden {
		t.Errorf("expected status %d for missing role context, got %d", http.StatusForbidden, w1.Code)
	}

	// Case 2: Member role (not allowed)
	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctx2 := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	ctx2 = contextWithValue(ctx2, ContextKeyRole, models.RoleMember)
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected status %d for member role, got %d", http.StatusForbidden, w2.Code)
	}

	// Case 3: Trainer role (allowed)
	req3 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctx3 := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	ctx3 = contextWithValue(ctx3, ContextKeyRole, models.RoleTrainer)
	req3 = req3.WithContext(ctx3)
	w3 := httptest.NewRecorder()
	mw.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("expected status %d for trainer role, got %d", http.StatusOK, w3.Code)
	}
}

func contextWithValue(parent context.Context, key contextKey, val any) context.Context {
	return context.WithValue(parent, key, val)
}
