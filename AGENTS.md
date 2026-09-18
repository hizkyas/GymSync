# GymOS — Agent Context & Project History

> **Purpose:** This file provides complete context for future AI agents working on GymOS (GymSync). It documents project architecture, decisions made, completed features, repository workflows, and current state. Read this document before making code modifications.

---

## 1. Project Overview & Architecture

**GymOS (GymSync)** is a multi-tenant gym membership and access control management application.

- **Backend:** Go 1.22 (`backend/`)
  - Framework: `go-chi/chi/v5`
  - Database: PostgreSQL (via `jackc/pgx/v5`)
  - Cache / Rate Limiting: Redis (`redis/v8`)
  - Authentication: JWT with RBAC (`admin`, `trainer`, `member`)
  - Payment & Subscriptions: Stripe SDK
- **Frontend Admin:** React 18 + Vite + TypeScript (`frontend-admin/`)
- **Orchestration:** Docker Compose (`docker-compose.yml`)

---

## 2. Git & Repository Rules

- **Repository Remote:** `git@github.com:hizkyas/GymSync.git`
- **Branch Strategy:** Feature Branch Workflow (PR-driven)
- **CRITICAL RULE:** **NEVER** commit or push directly to `main`. Always create a feature branch (`feature/GYM-xxx-...`) and submit a Pull Request.

---

## 3. Completed Work History

### Phase 0: Project Scaffold (`commit 6a3ef9a`)
- Scaffolding of Go backend (`cmd/api`, `internal/database`, `internal/handlers`, `internal/middleware`, `internal/models`).
- Postgres migrations schema (`users`, `membership_plans`, `subscriptions`, `check_ins`).
- Docker Compose configuration and React admin frontend scaffold.

### Phase 1: GYM-002 — QR Check-in Engine (`commit 8a57e88`)
- **Branch:** `feature/GYM-002-qr-checkin-engine` (PR #1: `https://github.com/hizkyas/GymSync/pull/1`)
- Introduced `DBQuerier` interface on `CheckInHandler`.
- Pure function `evaluateAccess()` handling all 5 subscription states.
- 19 test cases passing.

### Phase 2: GYM-003 — Stripe Webhook & Subscription Management (`commit 4cb1ecc`)
- **Branch:** `feature/GYM-003-stripe-webhook-subscriptions` (Pushed to remote `origin`)
- **Files Modified / Created:**
  - `backend/internal/handlers/subscription.go` — Introduced `SubscriptionDB` interface, added `invoice.payment_failed` and `invoice.payment_succeeded` handlers.
  - `backend/internal/handlers/subscription_test.go` — Added 9 test cases covering `mapStripeStatus`, dev/prod webhook signature logic, DB exec results, and error paths.
- **Test Results:** 9/9 handler tests passing (`go test ./internal/handlers/...`).

---

## 4. Current Repository State

- `main` is up-to-date with remote `origin/main`.
- `feature/GYM-002-qr-checkin-engine` PR #1 created.
- `feature/GYM-003-stripe-webhook-subscriptions` pushed to remote `origin`.

---

## 5. Next Steps & Ongoing Backlog

1. Open PR for `feature/GYM-003-stripe-webhook-subscriptions`.
2. Next production task: GYM-004 JWT Auth & RBAC Middleware Test Suite or GYM-005 Admin Dashboard Frontend integration.
