package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/handlers"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// newCheckInRequest builds an HTTP POST request with the given JSON body.
func newCheckInRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkin/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// decodeResponse decodes a JSON response body into a map for assertion.
func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// newMock creates a pgxmock pool and returns the handler built from it.
func newMock(t *testing.T) (pgxmock.PgxPoolIface, *handlers.CheckInHandler) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pgxmock pool: %v", err)
	}
	t.Cleanup(func() { mock.Close() })
	return mock, handlers.NewCheckInHandlerWithPool(mock)
}

// ─── Unit tests: evaluateAccess ───────────────────────────────────────────────
// EvaluateAccess is a pure function — no DB needed.

// TestEvaluateAccess_NilStatus verifies a nil subscription denies access.
func TestEvaluateAccess_NilStatus(t *testing.T) {
	t.Parallel()
	granted, reason := handlers.EvaluateAccess(nil)
	if granted {
		t.Fatal("expected access denied for nil subscription")
	}
	if reason == nil || *reason == "" {
		t.Fatal("expected non-empty denial reason")
	}
}

// TestEvaluateAccess_Active verifies an active subscription grants access.
func TestEvaluateAccess_Active(t *testing.T) {
	t.Parallel()
	status := models.StatusActive
	granted, reason := handlers.EvaluateAccess(&status)
	if !granted {
		t.Fatalf("expected access granted for active subscription, got reason: %v", reason)
	}
	if reason != nil {
		t.Fatalf("expected nil denial reason, got: %s", *reason)
	}
}

// TestEvaluateAccess_Trialing verifies a trialing subscription grants access.
func TestEvaluateAccess_Trialing(t *testing.T) {
	t.Parallel()
	status := models.StatusTrialing
	granted, _ := handlers.EvaluateAccess(&status)
	if !granted {
		t.Fatal("expected access granted for trialing subscription")
	}
}

// TestEvaluateAccess_PastDue verifies a past-due subscription denies access.
func TestEvaluateAccess_PastDue(t *testing.T) {
	t.Parallel()
	status := models.StatusPastDue
	granted, reason := handlers.EvaluateAccess(&status)
	if granted {
		t.Fatal("expected access denied for past_due subscription")
	}
	if reason == nil {
		t.Fatal("expected denial reason, got nil")
	}
	if !strings.Contains(strings.ToLower(*reason), "past due") {
		t.Fatalf("denial reason %q does not mention 'past due'", *reason)
	}
}

// TestEvaluateAccess_Canceled verifies a canceled subscription denies access.
func TestEvaluateAccess_Canceled(t *testing.T) {
	t.Parallel()
	status := models.StatusCanceled
	granted, reason := handlers.EvaluateAccess(&status)
	if granted {
		t.Fatal("expected access denied for canceled subscription")
	}
	if reason == nil {
		t.Fatal("expected denial reason, got nil")
	}
}

// TestEvaluateAccess_Paused verifies a paused subscription denies access.
func TestEvaluateAccess_Paused(t *testing.T) {
	t.Parallel()
	status := models.StatusPaused
	granted, reason := handlers.EvaluateAccess(&status)
	if granted {
		t.Fatal("expected access denied for paused subscription")
	}
	if reason == nil {
		t.Fatal("expected denial reason, got nil")
	}
}

// ─── Integration tests: CheckIn HTTP handler ──────────────────────────────────
// pgxmock stubs DB responses — no real Postgres required.

// TestCheckIn_InvalidBody verifies 400 on malformed JSON.
func TestCheckIn_InvalidBody(t *testing.T) {
	t.Parallel()
	_, h := newMock(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkin/", bytes.NewBufferString("{bad json}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// TestCheckIn_MissingToken verifies 400 when qr_token is empty.
func TestCheckIn_MissingToken(t *testing.T) {
	t.Parallel()
	_, h := newMock(t)

	req := newCheckInRequest(t, map[string]string{"qr_token": ""})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	body := decodeResponse(t, rr)
	if body["error"] == nil {
		t.Fatal("expected error field in response")
	}
}

// TestCheckIn_InvalidUUID verifies 400 on a malformed QR token UUID.
func TestCheckIn_InvalidUUID(t *testing.T) {
	t.Parallel()
	_, h := newMock(t)

	req := newCheckInRequest(t, map[string]string{"qr_token": "not-a-uuid"})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// TestCheckIn_MemberNotFound verifies 404 when the QR token matches no user.
func TestCheckIn_MemberNotFound(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: body=%s", rr.Code, rr.Body.String())
	}
}

// TestCheckIn_DeactivatedMember verifies 403 when the user's account is inactive.
func TestCheckIn_DeactivatedMember(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusActive

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "John Doe", "john@example.com", nil, false, &status,
		))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

// TestCheckIn_AccessGranted_Active verifies 200 for an active member.
func TestCheckIn_AccessGranted_Active(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusActive

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Jane Gym", "jane@gym.com", nil, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := decodeResponse(t, rr)
	if body["access_granted"] != true {
		t.Fatalf("expected access_granted=true, got: %v", body["access_granted"])
	}
	if body["member_name"] != "Jane Gym" {
		t.Fatalf("expected member_name='Jane Gym', got: %v", body["member_name"])
	}
	if body["denial_reason"] != nil {
		t.Fatalf("expected no denial_reason, got: %v", body["denial_reason"])
	}
}

// TestCheckIn_AccessGranted_Trialing verifies 200 for a trialing member.
func TestCheckIn_AccessGranted_Trialing(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusTrialing

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Trial Tara", "tara@gym.com", nil, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decodeResponse(t, rr)
	if body["access_granted"] != true {
		t.Fatalf("expected access_granted=true for trialing")
	}
}

// TestCheckIn_AccessDenied_PastDue verifies 403 for a past_due subscription.
func TestCheckIn_AccessDenied_PastDue(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusPastDue

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Past Due Pete", "pete@gym.com", nil, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	body := decodeResponse(t, rr)
	if body["access_granted"] != false {
		t.Fatal("expected access_granted=false")
	}
	if body["denial_reason"] == nil {
		t.Fatal("expected denial_reason to be set")
	}
}

// TestCheckIn_AccessDenied_Canceled verifies 403 for a canceled subscription.
func TestCheckIn_AccessDenied_Canceled(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusCanceled

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Canceled Carl", "carl@gym.com", nil, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

// TestCheckIn_AccessDenied_Paused verifies 403 for a paused subscription.
func TestCheckIn_AccessDenied_Paused(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusPaused

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Paused Paul", "paul@gym.com", nil, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

// TestCheckIn_AccessDenied_NoSubscription verifies 403 when subscription is nil.
func TestCheckIn_AccessDenied_NoSubscription(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "New Member", "new@gym.com", nil, true,
			(*models.SubscriptionStatus)(nil),
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	body := decodeResponse(t, rr)
	if body["access_granted"] != false {
		t.Fatal("expected access_granted=false")
	}
}

// TestCheckIn_ResponseFields verifies all required fields are present in response.
func TestCheckIn_ResponseFields(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusActive
	avatar := "https://example.com/avatar.jpg"

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Field Fiona", "fiona@gym.com", &avatar, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String()})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := decodeResponse(t, rr)

	for _, field := range []string{"access_granted", "member_name", "member_email", "checked_in_at", "check_in_id"} {
		if _, ok := body[field]; !ok {
			t.Errorf("response missing required field: %s", field)
		}
	}

	if body["avatar_url"] != avatar {
		t.Errorf("expected avatar_url=%q, got %v", avatar, body["avatar_url"])
	}

	checkInIDStr, ok := body["check_in_id"].(string)
	if !ok {
		t.Fatal("check_in_id is not a string")
	}
	if _, err := uuid.Parse(checkInIDStr); err != nil {
		t.Fatalf("check_in_id %q is not a valid UUID: %v", checkInIDStr, err)
	}

	checkedInAtStr, ok := body["checked_in_at"].(string)
	if !ok {
		t.Fatal("checked_in_at is not a string")
	}
	// Accept both RFC3339 and RFC3339Nano
	_, err1 := time.Parse(time.RFC3339, checkedInAtStr)
	_, err2 := time.Parse(time.RFC3339Nano, checkedInAtStr)
	if err1 != nil && err2 != nil {
		t.Fatalf("checked_in_at %q is not RFC3339", checkedInAtStr)
	}
}

// TestCheckIn_WithNotes verifies notes field is accepted in the request.
func TestCheckIn_WithNotes(t *testing.T) {
	t.Parallel()
	mock, h := newMock(t)

	token := uuid.New()
	memberID := uuid.New()
	status := models.StatusActive
	note := "Front desk manual override"

	cols := []string{"id", "full_name", "email", "avatar_url", "is_active", "status"}
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(
			memberID, "Notes Nina", "nina@gym.com", nil, true, &status,
		))
	mock.ExpectExec(`INSERT INTO check_ins`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req := newCheckInRequest(t, models.CheckInRequest{QRToken: token.String(), Notes: &note})
	rr := httptest.NewRecorder()
	h.CheckIn(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ─── Benchmark ────────────────────────────────────────────────────────────────

// BenchmarkEvaluateAccess measures pure access-evaluation throughput.
func BenchmarkEvaluateAccess(b *testing.B) {
	status := models.StatusActive
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handlers.EvaluateAccess(&status)
	}
}
