# Gymfusion — Agent Context & Project History

> **Purpose:** This file provides complete context for future AI agents working on Gymfusion. It documents project architecture, decisions made, completed features, repository workflows, and current state. Read this document before making code modifications.

---

## 1. Project Overview & Architecture

**Gymfusion** is a multi-tenant gym membership and access control management application.

- **Backend:** Go 1.22 (`backend/`)
  - Framework: `go-chi/chi/v5`
  - Database: PostgreSQL (via `jackc/pgx/v5`)
  - Cache / Rate Limiting: Redis (`redis/go-redis/v9`)
  - Authentication: JWT with RBAC (`admin`, `trainer`, `member`)
  - Payment & Subscriptions: Stripe SDK
- **Frontend Admin:** React 18 + Vite + TypeScript (`frontend-admin/`)
- **Orchestration:** Docker Compose (`docker-compose.yml`)

---

## 2. Git & Repository Rules

- **Repository Remote:** `git@github.com:hizkyas/Gymfusion.git`
- **Branch Strategy:** Feature Branch Workflow (PR-driven)
- **CRITICAL RULE:** **NEVER** commit or push directly to `main`. Always create a feature branch (`feature/GYM-xxx-...`) and submit a Pull Request.
- **Branch Protection:** `main` requires the `All Checks Passed` CI status check before any merge is allowed.

---

## 3. CI/CD Pipeline (GitHub Actions)

All three workflows live in `.github/workflows/`:

| Workflow | Trigger | What it enforces |
|---|---|---|
| `backend-ci.yml` | Push/PR to `backend/**` | `go build`, `go vet`, `go test -race ./...` ≥ 60% total coverage |
| `frontend-ci.yml` | Push/PR to `frontend-admin/**` | `oxlint`, `tsc --noEmit`, `vite build` |
| `pr-quality-gate.yml` | All PRs targeting `main` | Runs both backend + frontend jobs in parallel; `All Checks Passed` gate |

**Coverage baseline (as of GYM-008):**

| Package | Coverage |
|---|---|
| `internal/handlers` | 90.5% |
| `internal/middleware` | 98.4% |
| `total (./...)` | **65.4%** |

---

## 4. Completed Work History

### GYM-001 — Project Scaffold & Docker Compose
- Go backend skeleton, PostgreSQL + Redis containers, env config loading.

### GYM-002 — QR Check-in Engine
- `CheckInHandler` with sliding-window duplicate prevention.
- `internal/handlers/checkin.go`, `models/checkin.go`.

### GYM-003 — Stripe Webhook & Subscription Management
- Stripe webhook validation, `upsertSubscription`, `cancelSubscription`, `updateInvoiceStatus`.
- `internal/handlers/subscription.go`.

### GYM-004 — JWT Auth & RBAC Middleware Test Suite
- `JWTAuthMiddleware`, `RequireRole`, `GenerateToken`, `GetUserID`, `GetRole`.
- `internal/middleware/auth.go` + `auth_test.go` (full 96.7–100% coverage).

### GYM-005 — React Admin Dashboard Integration
- Vite + React 18 + TypeScript frontend scaffold.
- Dashboard, Login, Check-in, Members, Subscriptions pages.
- React Query data fetching, Axios API client.
- `frontend-admin/src/` — full page set with sidebar navigation.

### GYM-006 — Redis Rate Limiter Test Suite
- Extracted `SlidingWindowLimiter(rdb, limit, keyPrefix, window)` from `RateLimiter` and `CheckInRateLimiter`.
- Introduced `RateLimitRedis` interface for test-time injection (miniredis/real Redis).
- `internal/middleware/ratelimit.go` refactor + `ratelimit_test.go` (11 tests, 98.4% coverage).
- Dependencies: `github.com/alicebob/miniredis/v2`.

### GYM-007 — GitHub Actions CI/CD Pipeline
- `.github/workflows/backend-ci.yml` — Go build, vet, test, coverage gate.
- `.github/workflows/frontend-ci.yml` — oxlint, tsc, vite build.
- `.github/workflows/pr-quality-gate.yml` — parallel jobs + `All Checks Passed` gate.
- Fixed: scoped test command + threshold to exclude untestable `cmd/api`, `config`, `database` packages.
- Fixed: `App.tsx` lazy `useState` auth init (removed `setState` in `useEffect` oxlint warning).

### GYM-008 — Rename to Gymfusion
- Repo renamed from `GymSync` → `Gymfusion` on GitHub.
- Filesystem: `/home/hizkyas/Documents/GymSync` → `/home/hizkyas/Documents/Gymfusion`.
- All source references updated: frontend titles, API service names, README, AGENTS.md.
- LocalStorage token key: `gym_token` → `gymfusion_token`.
- Added 14 new handler tests (`checkin_admin_test.go`) covering `RecentCheckIns`, `Stats`, `Members`, `realClientIP`.
- Total coverage raised from 35.5% → **65.4%** (satisfies 60% CI gate).

---

## 5. Key Design Decisions

### Rate Limiter Architecture
`SlidingWindowLimiter` is the testable core. Both `RateLimiter` (60 req/min) and `CheckInRateLimiter` (10 req/min) are thin wrappers that delegate to it. The `RateLimitRedis` interface allows injecting `miniredis` in tests.

### Test Strategy
- **Handlers:** `pgxmock/v3` via the `DBQuerier` interface. `NewCheckInHandlerWithPool(mock)` exposed through `checkin_export_test.go`.
- **Middleware:** `miniredis/v2` for rate limiter integration tests; `httptest.NewRecorder` + JWT fixtures for auth middleware.
- **Coverage gate:** Applied to `./...` (all packages). `cmd/api`, `config`, `database` packages are excluded from the coverage calculation because they require live infrastructure — their 0% is intentional and expected.

### Frontend Auth State
Auth state is initialized with a lazy `useState(() => !!localStorage.getItem('gymfusion_token'))` instead of `useEffect`. This avoids a double-render and satisfies the `set-state-in-effect` oxlint rule.

---

## 6. Current State & Next Steps

**All PRs merged into `main`** as of GYM-008. The project is stable and CI-green.

Suggested next features:
- `GYM-010` — Member self-service portal (React Native or separate web app)
- `GYM-011` — Admin user management (create/deactivate members)
- `GYM-012` — Reporting & analytics dashboard (charts, export CSV)
- `GYM-013` — Docker Compose production hardening (nginx, TLS, secrets)
- `GYM-014` — E2E integration tests (testcontainers-go)

---

## 7. File Map (Key Files)

```
Gymfusion/
├── backend/
│   ├── cmd/api/main.go                     — entrypoint, router wiring
│   ├── internal/
│   │   ├── config/config.go                — env loading
│   │   ├── database/postgres.go redis.go   — connection pools
│   │   ├── handlers/
│   │   │   ├── auth.go auth_test.go        — register/login/me
│   │   │   ├── checkin.go checkin_test.go  — QR check-in + admin reads
│   │   │   ├── checkin_admin_test.go       — RecentCheckIns/Stats/Members tests
│   │   │   ├── checkin_export_test.go      — white-box test helpers
│   │   │   ├── subscription.go sub_test.go — Stripe webhook
│   │   │   └── helpers.go                  — respondJSON/respondError
│   │   ├── middleware/
│   │   │   ├── auth.go auth_test.go        — JWT + RBAC
│   │   │   └── ratelimit.go ratelimit_test.go — sliding window rate limits
│   │   └── models/                         — Go structs (User, CheckIn, etc.)
│   └── go.mod go.sum
├── frontend-admin/
│   └── src/
│       ├── App.tsx                         — routes + auth state
│       ├── pages/                          — Dashboard, Login, CheckIn, Members
│       ├── components/                     — QRScanner, shared UI
│       └── services/api.ts                 — Axios client with JWT inject
└── .github/workflows/
    ├── backend-ci.yml
    ├── frontend-ci.yml
    └── pr-quality-gate.yml
```
