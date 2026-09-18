package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBQuerier is the minimal DB interface required by CheckInHandler.
// Both *pgxpool.Pool and pgxmock.PgxPoolIface satisfy this interface,
// making the handler fully testable without a real database.
type DBQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// CheckInHandler handles QR-code check-in requests.
type CheckInHandler struct {
	db DBQuerier
}

// NewCheckInHandler creates a CheckInHandler backed by a production pgxpool.
func NewCheckInHandler(db *pgxpool.Pool) *CheckInHandler {
	return &CheckInHandler{db: db}
}

// newCheckInHandlerFromQuerier creates a CheckInHandler from any DBQuerier.
// Used in tests to inject a mock.
func newCheckInHandlerFromQuerier(q DBQuerier) *CheckInHandler {
	return &CheckInHandler{db: q}
}

// CheckIn handles POST /api/v1/checkin
//
// Hot path — optimised for sub-10 ms response times.
// Strategy:
//  1. Parse and validate the QR token UUID from the request body.
//  2. Execute a single LATERAL JOIN query that resolves the member and their
//     most-recent subscription status in one round-trip.
//  3. Evaluate access: granted for `active`/`trialing`, denied otherwise.
//  4. Persist the access log asynchronously (does NOT block the response).
//  5. Return 200 OK (access granted) or 403 Forbidden (access denied).
func (h *CheckInHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	// ── 1. Parse request ────────────────────────────────────────────────────
	var req models.CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.QRToken = strings.TrimSpace(req.QRToken)
	if req.QRToken == "" {
		respondError(w, http.StatusBadRequest, "qr_token is required")
		return
	}

	qrToken, err := uuid.Parse(req.QRToken)
	if err != nil {
		respondError(w, http.StatusBadRequest, "qr_token must be a valid UUID")
		return
	}

	// ── 2. DB lookup ─────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID, fullName, email, avatarURL, isActive, subStatus, err :=
		h.lookupMember(ctx, qrToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "no member found for this QR token")
			return
		}
		respondError(w, http.StatusInternalServerError, "lookup failed, please try again")
		return
	}

	// ── 3. Account active guard ──────────────────────────────────────────────
	if !isActive {
		respondError(w, http.StatusForbidden, "member account is deactivated")
		return
	}

	// ── 4. Evaluate subscription status ─────────────────────────────────────
	accessGranted, denialReason := evaluateAccess(subStatus)

	// ── 5. Persist access log (async) ────────────────────────────────────────
	checkInID := uuid.New()
	checkedInAt := time.Now().UTC()
	ipStr := realClientIP(r)

	go h.persistCheckIn(checkInID, userID, qrToken, accessGranted, denialReason, checkedInAt, ipStr, req.Notes)

	// ── 6. Respond ───────────────────────────────────────────────────────────
	resp := models.CheckInResponse{
		AccessGranted: accessGranted,
		DenialReason:  denialReason,
		MemberName:    fullName,
		MemberEmail:   email,
		AvatarURL:     avatarURL,
		CheckedInAt:   checkedInAt,
		CheckInID:     checkInID,
	}

	statusCode := http.StatusOK
	if !accessGranted {
		statusCode = http.StatusForbidden
	}
	respondJSON(w, statusCode, resp)
}

// lookupMember performs the single optimised query: user row + lateral
// subscription join, all resolved in one DB round-trip.
func (h *CheckInHandler) lookupMember(
	ctx context.Context,
	qrToken uuid.UUID,
) (
	userID uuid.UUID,
	fullName, email string,
	avatarURL *string,
	isActive bool,
	subStatus *models.SubscriptionStatus,
	err error,
) {
	err = h.db.QueryRow(ctx, `
		SELECT
			u.id,
			u.full_name,
			u.email,
			u.avatar_url,
			u.is_active,
			s.status
		FROM users u
		LEFT JOIN LATERAL (
			SELECT status
			FROM subscriptions
			WHERE user_id = u.id
			ORDER BY created_at DESC
			LIMIT 1
		) s ON true
		WHERE u.qr_token = $1
	`, qrToken).Scan(
		&userID, &fullName, &email, &avatarURL, &isActive, &subStatus,
	)
	return
}

// evaluateAccess determines whether a member should be granted access based
// on their current subscription status.
//
// Access is GRANTED for:
//   - active   — paid and within the billing period
//   - trialing — within a free trial period
//
// Access is DENIED for all other statuses, each with a descriptive reason.
func evaluateAccess(subStatus *models.SubscriptionStatus) (accessGranted bool, denialReason *string) {
	if subStatus == nil {
		reason := "no subscription found — please sign up for a membership plan"
		return false, &reason
	}

	switch *subStatus {
	case models.StatusActive, models.StatusTrialing:
		return true, nil

	case models.StatusPastDue:
		reason := "subscription payment is past due — please update your payment method"
		return false, &reason

	case models.StatusCanceled:
		reason := "subscription has been canceled — please renew to regain access"
		return false, &reason

	case models.StatusPaused:
		reason := "subscription is currently paused — please resume your plan"
		return false, &reason

	default:
		reason := "subscription is not in an active state"
		return false, &reason
	}
}

// persistCheckIn writes the access log row to check_ins.
// Runs in a background goroutine to keep the hot path non-blocking.
func (h *CheckInHandler) persistCheckIn(
	checkInID, userID, qrToken uuid.UUID,
	accessGranted bool,
	denialReason *string,
	checkedInAt time.Time,
	ipStr string,
	notes *string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, _ = h.db.Exec(ctx, `
		INSERT INTO check_ins
			(id, user_id, qr_token, access_granted, denial_reason, checked_in_at, ip_address, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8)
	`, checkInID, userID, qrToken, accessGranted, denialReason, checkedInAt, ipStr, notes)
}

// ─── Supporting endpoints ─────────────────────────────────────────────────────

// RecentCheckIns handles GET /api/v1/checkins/recent (admin/trainer only)
func (h *CheckInHandler) RecentCheckIns(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT
			ci.id,
			u.full_name,
			u.email,
			u.avatar_url,
			ci.access_granted,
			ci.denial_reason,
			COALESCE(s.status, 'canceled') AS status,
			ci.checked_in_at
		FROM check_ins ci
		JOIN users u ON u.id = ci.user_id
		LEFT JOIN LATERAL (
			SELECT status FROM subscriptions WHERE user_id = ci.user_id
			ORDER BY created_at DESC LIMIT 1
		) s ON true
		ORDER BY ci.checked_in_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch recent check-ins")
		return
	}
	defer rows.Close()

	var result []models.RecentCheckIn
	for rows.Next() {
		var ci models.RecentCheckIn
		if err := rows.Scan(
			&ci.CheckInID, &ci.MemberName, &ci.MemberEmail, &ci.AvatarURL,
			&ci.AccessGranted, &ci.DenialReason, &ci.Status, &ci.CheckedInAt,
		); err != nil {
			continue
		}
		result = append(result, ci)
	}
	if result == nil {
		result = []models.RecentCheckIn{}
	}

	respondJSON(w, http.StatusOK, result)
}

// Stats handles GET /api/v1/stats (admin only)
func (h *CheckInHandler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	type statsPayload struct {
		ActiveMembers  int64   `json:"active_members"`
		TotalMembers   int64   `json:"total_members"`
		TodayCheckIns  int64   `json:"today_check_ins"`
		MonthlyRevenue float64 `json:"monthly_revenue_cents"`
	}

	var s statsPayload

	_ = h.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE s.status = 'active') AS active_members,
			COUNT(*) AS total_members
		FROM users u
		LEFT JOIN LATERAL (
			SELECT status FROM subscriptions WHERE user_id = u.id
			ORDER BY created_at DESC LIMIT 1
		) s ON true
		WHERE u.role = 'member'
	`).Scan(&s.ActiveMembers, &s.TotalMembers)

	_ = h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE checked_in_at >= CURRENT_DATE
	`).Scan(&s.TodayCheckIns)

	_ = h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(p.price_cents), 0)
		FROM subscriptions sub
		JOIN membership_plans p ON p.id = sub.plan_id
		WHERE sub.status = 'active'
	`).Scan(&s.MonthlyRevenue)

	respondJSON(w, http.StatusOK, s)
}

// Members handles GET /api/v1/members (admin/trainer only)
func (h *CheckInHandler) Members(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT
			u.id, u.email, u.full_name, u.role, u.phone, u.avatar_url,
			u.qr_token, u.is_active, u.created_at, u.updated_at,
			s.status
		FROM users u
		LEFT JOIN LATERAL (
			SELECT status FROM subscriptions WHERE user_id = u.id
			ORDER BY created_at DESC LIMIT 1
		) s ON true
		WHERE u.role = 'member'
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch members")
		return
	}
	defer rows.Close()

	type memberRow struct {
		models.User
		SubscriptionStatus *string `json:"subscription_status"`
	}

	var members []memberRow
	for rows.Next() {
		var m memberRow
		if err := rows.Scan(
			&m.ID, &m.Email, &m.FullName, &m.Role, &m.Phone, &m.AvatarURL,
			&m.QRToken, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
			&m.SubscriptionStatus,
		); err != nil {
			continue
		}
		members = append(members, m)
	}
	if members == nil {
		members = []memberRow{}
	}

	respondJSON(w, http.StatusOK, members)
}

// realClientIP extracts the best-guess client IP from the request.
func realClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
