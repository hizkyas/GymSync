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
- **Branch:** `feature/GYM-002-qr-checkin-engine` (PR #1)
- Introduced `DBQuerier` interface on `CheckInHandler`.
- Pure function `evaluateAccess()` handling all 5 subscription states.
- 19 test cases passing.

### Phase 2: GYM-003 — Stripe Webhook & Subscription Management (`commit 4cb1ecc` / `72c59fe`)
- **Branch:** `feature/GYM-003-stripe-webhook-subscriptions` (PR #2)
- Introduced `SubscriptionDB` interface.
- Added event handlers for `invoice.payment_failed` and `invoice.payment_succeeded`.
- 9 test cases passing.

### Phase 3: GYM-004 — JWT Auth & RBAC Middleware Test Suite (`commit 9755566` / `c92152d`)
- **Branch:** `feature/GYM-004-jwt-auth-rbac-suite`
- Introduced `AuthDB` interface on `AuthHandler`.
- Added 14 unit & integration test cases (`handlers/auth_test.go` and `middleware/auth_test.go`).

### Phase 4: GYM-005 — React Admin Dashboard Integration (`commit a910e5d`)
- **Branch:** `feature/GYM-005-admin-dashboard-integration` (Pushed to remote `origin`)
- Integrated `@tanstack/react-query` `useQueryClient` cache invalidation in `QRScanner.tsx` for immediate feed & stats update.
- Verified TypeScript build & production asset bundle generation (`npm run build`).

---

## 4. Current Repository State

- `main` is up-to-date with remote `origin/main`.
- `feature/GYM-002-qr-checkin-engine` (PR #1 open).
- `feature/GYM-003-stripe-webhook-subscriptions` (PR #2 open).
- `feature/GYM-004-jwt-auth-rbac-suite` pushed.
- `feature/GYM-005-admin-dashboard-integration` pushed.
