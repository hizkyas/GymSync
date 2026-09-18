package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckInHandler handles QR-code check-in requests.
type CheckInHandler struct {
	db *pgxpool.Pool
}

// NewCheckInHandler creates a new CheckInHandler.
func NewCheckInHandler(db *pgxpool.Pool) *CheckInHandler {
	return &CheckInHandler{db: db}
}

// CheckIn handles POST /api/v1/checkin
// This is the hot path — optimized for sub-10ms response times.
func (h *CheckInHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req models.CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	qrToken, err := uuid.Parse(req.QRToken)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid qr_token format")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Single optimized query: lookup user + latest subscription status via index on qr_token
	var (
		userID        uuid.UUID
		fullName      string
		email         string
		avatarURL     *string
		isActive      bool
		subStatus     *models.SubscriptionStatus
	)

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
	`, qrToken).Scan(&userID, &fullName, &email, &avatarURL, &isActive, &subStatus)
	if err != nil {
		respondError(w, http.StatusNotFound, "member not found for this QR token")
		return
	}

	if !isActive {
		respondError(w, http.StatusForbidden, "member account is deactivated")
		return
	}

	// Determine access
	accessGranted := false
	var denialReason *string

	if subStatus == nil {
		reason := "no active subscription found"
		denialReason = &reason
	} else {
		switch *subStatus {
		case models.StatusActive, models.StatusTrialing:
			accessGranted = true
		case models.StatusPastDue:
			reason := "subscription payment is past due"
			denialReason = &reason
		case models.StatusCanceled:
			reason := "subscription has been canceled"
			denialReason = &reason
		case models.StatusPaused:
			reason := "subscription is currently paused"
			denialReason = &reason
		default:
			reason := "subscription is not active"
			denialReason = &reason
		}
	}

	// Extract client IP
	ipStr := realClientIP(r)

	// Write access log asynchronously to avoid blocking the response
	checkInID := uuid.New()
	checkedInAt := time.Now().UTC()

	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer bgCancel()
		_, _ = h.db.Exec(bgCtx, `
			INSERT INTO check_ins (id, user_id, qr_token, access_granted, denial_reason, checked_in_at, ip_address, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8)
		`, checkInID, userID, qrToken, accessGranted, denialReason, checkedInAt, ipStr, req.Notes)
	}()

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

	type stats struct {
		ActiveMembers  int64   `json:"active_members"`
		TotalMembers   int64   `json:"total_members"`
		TodayCheckIns  int64   `json:"today_check_ins"`
		MonthlyRevenue float64 `json:"monthly_revenue_cents"`
	}

	var s stats

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
