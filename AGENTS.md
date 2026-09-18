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
- **Branch:** `feature/GYM-002-qr-checkin-engine` (Pushed to remote `origin`)
- **Files Modified:**
  - `backend/internal/handlers/checkin.go` — Handler implementation for `POST /api/v1/checkin`.
  - `backend/internal/handlers/checkin_export_test.go` — White-box testing export bridge.
  - `backend/internal/handlers/checkin_test.go` — 19 test cases (unit, mock integration, benchmarks).
  - `backend/go.mod` & `backend/go.sum` — Added `github.com/pashagolub/pgxmock/v3`.
- **Key Technical Implementation:**
  - `DBQuerier` interface on `CheckInHandler` to enable mock DB testing without live Postgres.
  - Pure function `evaluateAccess()` handling all 5 subscription states (`active`, `trialing`, `past_due`, `canceled`, `paused`, `nil`).
  - Strict payload validation (trimmed whitespace, token present, valid UUID).
  - Asynchronous non-blocking check-in logging (`persistCheckIn` goroutine).
  - 19/19 tests passing (`go test ./internal/handlers/...`).

---

## 4. Current Repository State

- `main` is up-to-date with remote `origin/main`.
- `feature/GYM-002-qr-checkin-engine` is up-to-date with remote `origin/feature/GYM-002-qr-checkin-engine`.
- PR URL for GYM-002: `https://github.com/hizkyas/GymSync/pull/new/feature/GYM-002-qr-checkin-engine`

---

## 5. Next Steps & Ongoing Backlog

1. Open PR for `feature/GYM-002-qr-checkin-engine` and merge into `main` after review.
2. Next production tasks (e.g. GYM-003 Stripe Webhook sync, Membership Management API, Admin Dashboard UI integration).
