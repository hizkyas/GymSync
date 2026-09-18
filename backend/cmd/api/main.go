package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hizkyas/gym-app/backend/internal/config"
	"github.com/hizkyas/gym-app/backend/internal/database"
	"github.com/hizkyas/gym-app/backend/internal/handlers"
	"github.com/hizkyas/gym-app/backend/internal/middleware"
	"github.com/hizkyas/gym-app/backend/internal/models"
	"github.com/stripe/stripe-go/v78"
)

func main() {
	// ─── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ─── Database connections ───────────────────────────────────────────────────
	ctx := context.Background()

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()
	log.Println("✅ PostgreSQL connected")

	rdb, err := database.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer rdb.Close()
	log.Println("✅ Redis connected")

	// ─── Stripe ─────────────────────────────────────────────────────────────────
	if cfg.StripeSecretKey != "" && cfg.StripeSecretKey != "sk_test_placeholder" {
		stripe.Key = cfg.StripeSecretKey
		log.Println("✅ Stripe configured")
	} else {
		log.Println("⚠️  Stripe not configured — webhook handler will run in dev mode")
	}

	// ─── Handlers ────────────────────────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret, cfg.JWTExpiryHours)
	checkInHandler := handlers.NewCheckInHandler(db)
	subHandler := handlers.NewSubscriptionHandler(db, cfg.StripeWebhookSecret)

	// ─── Router ──────────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Global rate limiter
	r.Use(middleware.RateLimiter(rdb))

	// ─── Routes ─────────────────────────────────────────────────────────────────

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok","service":"gym-api"}`)
	})

	// Stripe webhook (Stripe signs it, no JWT needed)
	r.Post("/api/v1/webhooks/stripe", subHandler.StripeWebhook)

	// Public auth routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
			r.Get("/me", authHandler.Me)
		})
	})

	// Check-in route (front-desk, stricter rate limit)
	r.Route("/api/v1/checkin", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
		r.Use(middleware.RequireRole(models.RoleAdmin, models.RoleTrainer))
		r.Use(middleware.CheckInRateLimiter(rdb))
		r.Post("/", checkInHandler.CheckIn)
	})

	// Protected API routes (admin + trainer)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

		// Recent check-ins — admin and trainer can view
		r.With(middleware.RequireRole(models.RoleAdmin, models.RoleTrainer)).
			Get("/checkins/recent", checkInHandler.RecentCheckIns)

		// Stats — admin only
		r.With(middleware.RequireRole(models.RoleAdmin)).
			Get("/stats", checkInHandler.Stats)

		// Members list — admin and trainer
		r.With(middleware.RequireRole(models.RoleAdmin, models.RoleTrainer)).
			Get("/members", checkInHandler.Members)
	})

	// ─── Server ──────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("🚀 Gym API server listening on :%s", cfg.Port)
		serverErr <- srv.ListenAndServe()
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case sig := <-quit:
		log.Printf("received signal %v, shutting down gracefully...", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("graceful shutdown failed: %v", err)
		}
		log.Println("server shut down cleanly")
	}
}
