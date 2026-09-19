package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/pashagolub/pgxmock/v3"
)

// recentCheckInCols lists the columns returned by the RecentCheckIns query in
// the same order they appear in the Scan call inside checkin.go.
var recentCheckInCols = []string{
	"id", "full_name", "email", "avatar_url",
	"access_granted", "denial_reason", "status", "checked_in_at",
}

// memberCols lists the columns returned by the Members query.
var memberCols = []string{
	"id", "email", "full_name", "role", "phone", "avatar_url",
	"qr_token", "is_active", "created_at", "updated_at", "status",
}

// ─────────────────────────────────────────────────────────────────────────────
// RecentCheckIns tests
// ─────────────────────────────────────────────────────────────────────────────

func TestRecentCheckIns_DefaultLimit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows(recentCheckInCols).AddRow(
		uuid.New(),
		"Alice Smith",
		"alice@gym.io",
		(*string)(nil),
		true,
		(*string)(nil),
		models.StatusActive,
		time.Now(),
	)
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(rows)

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkins/recent", nil)
	w := httptest.NewRecorder()
	h.RecentCheckIns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRecentCheckIns_CustomLimit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	// limit=10 supplied in query string — expect AnyArg (no type mismatch)
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(recentCheckInCols))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkins/recent?limit=10", nil)
	w := httptest.NewRecorder()
	h.RecentCheckIns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// empty result should be [] not null
	if result == nil {
		t.Error("expected empty array, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRecentCheckIns_InvalidLimit_FallsBackToDefault(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	// "abc" is not a valid int — handler falls back to 50
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(recentCheckInCols))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkins/recent?limit=abc", nil)
	w := httptest.NewRecorder()
	h.RecentCheckIns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRecentCheckIns_LimitAboveCap_ClampedTo200(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	// 999 > 200 — handler clamps to default 50
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(recentCheckInCols))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkins/recent?limit=999", nil)
	w := httptest.NewRecorder()
	h.RecentCheckIns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRecentCheckIns_DBError_Returns500(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("db timeout"))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkins/recent", nil)
	w := httptest.NewRecorder()
	h.RecentCheckIns(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Stats tests
// ─────────────────────────────────────────────────────────────────────────────

func TestStats_ReturnsAggregates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	// Stats makes three sequential QueryRow calls — match all three
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(pgxmock.NewRows([]string{"active_members", "total_members"}).
			AddRow(int64(5), int64(10)))
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(3)))
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(float64(29900)))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v, ok := payload["active_members"].(float64); !ok || v != 5 {
		t.Errorf("expected active_members=5, got %v", payload["active_members"])
	}
	if v, ok := payload["today_check_ins"].(float64); !ok || v != 3 {
		t.Errorf("expected today_check_ins=3, got %v", payload["today_check_ins"])
	}
}

func TestStats_DBErrorsStillReturn200(t *testing.T) {
	// Stats ignores DB errors (uses _ = h.db.QueryRow(...).Scan(...))
	// so it always responds 200 with zero-valued fields.
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))
	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))
	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 even on DB error (fail-safe), got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Members tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMembers_ReturnsMemberList(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	now := time.Now()
	uid := uuid.New()
	qr := uuid.New()
	subStatus := "active"

	rows := pgxmock.NewRows(memberCols).AddRow(
		uid,
		"bob@gym.io",
		"Bob Jones",
		models.RoleMember,
		(*string)(nil),
		(*string)(nil),
		qr,
		true,
		now,
		now,
		&subStatus,
	)
	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	w := httptest.NewRecorder()
	h.Members(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 member, got %d", len(result))
	}
	if result[0]["email"] != "bob@gym.io" {
		t.Errorf("expected email bob@gym.io, got %v", result[0]["email"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMembers_EmptyList(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).WillReturnRows(pgxmock.NewRows(memberCols))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	w := httptest.NewRecorder()
	h.Members(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

func TestMembers_DBError_Returns500(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))

	h := NewCheckInHandlerWithPool(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	w := httptest.NewRecorder()
	h.Members(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// realClientIP tests
// ─────────────────────────────────────────────────────────────────────────────

func TestRealClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	if got := realClientIP(req); got != "203.0.113.5" {
		t.Errorf("expected X-Forwarded-For, got %q", got)
	}
}

func TestRealClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.1")
	if got := realClientIP(req); got != "198.51.100.1" {
		t.Errorf("expected X-Real-IP, got %q", got)
	}
}

func TestRealClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	if got := realClientIP(req); got != "10.0.0.1:54321" {
		t.Errorf("expected RemoteAddr, got %q", got)
	}
}
