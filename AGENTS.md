# Gymfusion — Gym Membership Application

> **Purpose:** This file provides complete context for future AI agents working on Gymfusion. It documents project architecture, decisions made, completed features, repository workflows, and current state. Read this document before making code modifications.

---

## 1. Project Overview & Architecture

**Gymfusion** is a multi-tenant gym membership and access control management application.

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

- **Repository Remote:** `git@github.com:hizkyas/Gymfusion.git` *(renamed from GymSync)*
- **Branch Strategy:** Feature Branch Workflow (PR-driven)
- **CRITICAL RULE:** **NEVER** commit or push directly to `main`. Always create a feature branch (`feature/GYM-xxx-...`) and submit a Pull Request.

---

## 3. Completed Work History

### Phase 0: Project Scaffold (`commit 6a3ef9a`)
- Scaffolding of Go backend (`cmd/api`, `internal/database`, `internal/handlers`, `internal/middleware`, `internal/models`).
- Postgres migrations schema (`users`, `membership_plans`, `subscriptions`, `check_ins`).
- Docker Compose configuration and React admin frontend scaffold.

### Phase 1: GYM-002 — QR Check-in Engine (`commit 8a57e88`)
- **Branch:** `feature/GYM-002-qr-checkin-engine` (PR #1 — merged)
- Introduced `DBQuerier` interface on `CheckInHandler`.
- Pure function `evaluateAccess()` handling all 5 subscription states.
- 19 test cases passing.

### Phase 2: GYM-003 — Stripe Webhook & Subscription Management
- **Branch:** `feature/GYM-003-stripe-webhook-subscriptions` (PR #2 — merged)
- Introduced `SubscriptionDB` interface.
- Added event handlers for `invoice.payment_failed` and `invoice.payment_succeeded`.
- 9 test cases passing.

### Phase 3: GYM-004 — JWT Auth & RBAC Middleware Test Suite
- **Branch:** `feature/GYM-004-jwt-auth-rbac-suite` (PR #3 — merged)
- Introduced `AuthDB` interface on `AuthHandler`.
- Added 14 unit & integration test cases across `handlers/auth_test.go` and `middleware/auth_test.go`.

### Phase 4: GYM-005 — React Admin Dashboard Integration
- **Branch:** `feature/GYM-005-admin-dashboard-integration` (PR #4 — merged)
- Integrated React Query cache invalidation in `QRScanner.tsx`.
- Verified TypeScript build & production asset bundle.

### Phase 5: GYM-006 — Redis Rate Limiter Test Suite
- **Branch:** `feature/GYM-006-rate-limiter-test-suite`
- Extracted `SlidingWindowLimiter` with `RateLimitRedis` interface.
- 9 test cases using miniredis in-memory server.

### Phase 6: GYM-007 — CI/CD Pipeline
- **Branch:** `feature/GYM-007-ci-cd-pipeline`
- GitHub Actions: `backend-ci.yml`, `frontend-ci.yml`, `pr-quality-gate.yml`.
- Enforces: go build, go vet, race-detected tests, coverage ≥60%, oxlint, tsc, vite build.

### Phase 7: GYM-008 — App Rename to Gymfusion
- **Branch:** `feature/GYM-008-rename-to-gymfusion`
- Renamed app from GymOS/GymSync to **Gymfusion** across all frontend, backend, and documentation files.
- Remote repo renamed to `git@github.com:hizkyas/Gymfusion.git`.

---

## 4. Current Repository State

- All PRs #1–#4 merged into `main`.
- `feature/GYM-006-rate-limiter-test-suite` pushed to remote.
- `feature/GYM-007-ci-cd-pipeline` pushed to remote.
- `feature/GYM-008-rename-to-gymfusion` in progress.

---

## 5. Token Key

- Frontend localStorage key for JWT: `gymfusion_token`
- Backend health check service name: `gymfusion-api`
