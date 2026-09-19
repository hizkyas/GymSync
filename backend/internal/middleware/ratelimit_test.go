package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newMiniredis starts an in-memory Redis server and returns the client + cleanup fn.
func newMiniredis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return rdb, mr
}

// okHandler returns 200 and is used as the downstream "next" handler in all tests.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// realIP helper tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRealIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	if got := realIP(req); got != "203.0.113.1:12345" {
		t.Errorf("expected RemoteAddr, got %q", got)
	}
}

func TestRealIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	if got := realIP(req); got != "203.0.113.5" {
		t.Errorf("expected X-Forwarded-For, got %q", got)
	}
}

func TestRealIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Real-IP", "198.51.100.9")
	if got := realIP(req); got != "198.51.100.9" {
		t.Errorf("expected X-Real-IP, got %q", got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// SlidingWindowLimiter integration tests via miniredis
// ──────────────────────────────────────────────────────────────────────────────

func TestSlidingWindowLimiter_FirstRequest_Allowed(t *testing.T) {
	rdb, mr := newMiniredis(t)
	defer mr.Close()

	mw := SlidingWindowLimiter(rdb, 10, "test_limit", time.Minute)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:0"
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("expected X-RateLimit-Limit=10, got %q", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "9" {
		t.Errorf("expected X-RateLimit-Remaining=9, got %q", w.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestSlidingWindowLimiter_ExactLimit_StillAllowed(t *testing.T) {
	rdb, mr := newMiniredis(t)
	defer mr.Close()

	mw := SlidingWindowLimiter(rdb, 5, "test_limit", time.Minute)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:0"

	var lastCode int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)
		lastCode = w.Code
	}

	// The 5th request exactly at limit should still pass
	if lastCode != http.StatusOK {
		t.Errorf("expected 200 at exactly limit, got %d", lastCode)
	}
}

func TestSlidingWindowLimiter_Exceeded_Returns429(t *testing.T) {
	rdb, mr := newMiniredis(t)
	defer mr.Close()

	limit := int64(3)
	mw := SlidingWindowLimiter(rdb, limit, "test_limit", time.Minute)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:0"

	// Burn through the limit
	for i := int64(0); i < limit; i++ {
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)
	}

	// Next request should be rate-limited
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
	if w.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("expected X-RateLimit-Remaining=0, got %q", w.Header().Get("X-RateLimit-Remaining"))
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Error("expected JSON error body in 429 response")
	}
}

func TestSlidingWindowLimiter_DifferentIPs_Independent(t *testing.T) {
	rdb, mr := newMiniredis(t)
	defer mr.Close()

	limit := int64(2)
	mw := SlidingWindowLimiter(rdb, limit, "test_limit", time.Minute)(okHandler())

	// Exhaust limit for IP A
	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "10.0.0.10:0"
	for i := int64(0); i < limit+1; i++ {
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, reqA)
	}

	// IP B should still be allowed
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "10.0.0.20:0"
	wB := httptest.NewRecorder()
	mw.ServeHTTP(wB, reqB)

	if wB.Code != http.StatusOK {
		t.Errorf("IP B should be unaffected, expected 200 got %d", wB.Code)
	}
}

func TestSlidingWindowLimiter_XForwardedFor_UsedAsKey(t *testing.T) {
	rdb, mr := newMiniredis(t)
	defer mr.Close()

	limit := int64(2)
	mw := SlidingWindowLimiter(rdb, limit, "test_limit", time.Minute)(okHandler())

	// Exhaust limit via X-Forwarded-For header
	for i := int64(0); i < limit+1; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:0"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)
	}

	// Same X-Forwarded-For, should now be rate-limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for exhausted forwarded IP, got %d", w.Code)
	}
}

func TestSlidingWindowLimiter_WindowExpiry_Resets(t *testing.T) {
	rdb, mr := newMiniredis(t)
	defer mr.Close()

	limit := int64(2)
	window := 2 * time.Second
	mw := SlidingWindowLimiter(rdb, limit, "test_limit", window)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:0"

	// Exhaust limit
	for i := int64(0); i < limit+1; i++ {
		w := httptest.NewRecorder()
		mw.ServeHTTP(w, req)
	}
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 before window expires, got %d", w.Code)
	}

	// Fast-forward miniredis time past the window so entries expire
	mr.FastForward(3 * time.Second)

	// Now requests should be allowed again
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 after window reset, got %d", w2.Code)
	}
}
