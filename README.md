# GymOS — Gym Membership Application

A scalable, multi-tenant gym management platform built with **Go**, **PostgreSQL**, **Redis**, and a **React + Vite** admin dashboard.

---

## Tech Stack

| Layer      | Technology |
|------------|------------|
| Backend    | Go 1.22, Chi router, pgx/v5, go-redis/v9 |
| Auth       | JWT (HS256) + RBAC (`admin`, `trainer`, `member`) |
| Database   | PostgreSQL 16 with `golang-migrate` |
| Cache      | Redis 7 (rate limiting, session) |
| Payments   | Stripe (subscription webhooks) |
| Frontend   | React 18 + TypeScript + Vite + React Query |

---

## Quickstart

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose v2
- Go 1.22+ (for local backend development)
- Node.js 20+ (for local frontend development)

### 1. Clone & configure

```bash
git clone <your-repo-url>
cd gym-app

cp .env.example .env
# Edit .env and fill in your JWT_SECRET and optional Stripe keys
```

### 2. Start all services with Docker Compose

```bash
docker compose up --build
```

This spins up:
- **PostgreSQL** on `localhost:5432`
- **Redis** on `localhost:6379`
- **Go API** on `http://localhost:8080`
- **Admin Dashboard** on `http://localhost:5173`

### 3. Apply database migrations

```bash
# Install golang-migrate
brew install golang-migrate   # macOS
# or: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path ./backend/migrations \
        -database "postgres://gymuser:gympassword@localhost:5432/gymdb?sslmode=disable" \
        up
```

### 4. Create your first admin user

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@gym.com","password":"admin1234","full_name":"Admin User","role":"admin"}' | jq
```

### 5. Open the dashboard

Navigate to **http://localhost:5173** and log in with your admin credentials.

---

## API Reference

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Register a new user |
| POST | `/api/v1/auth/login` | Login and receive JWT |
| GET  | `/api/v1/auth/me` | Get current user (auth required) |

### Check-in
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/checkin/` | Verify QR token, log access (admin/trainer) |
| GET  | `/api/v1/checkins/recent` | Recent check-in feed (admin/trainer) |

### Management (admin only)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/stats` | Active members, MRR, daily check-ins |
| GET | `/api/v1/members` | List all members with subscription status |

### Webhooks
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/webhooks/stripe` | Stripe subscription event handler |

---

## Local Development

### Backend

```bash
cd backend
cp ../.env.example ../.env  # ensure .env exists

# Run with hot-reload (requires air)
go install github.com/air-verse/air@latest
air

# Or just:
go run ./cmd/api
```

### Frontend

```bash
cd frontend-admin
npm run dev
# Vite dev server on http://localhost:5173
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API server port |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `JWT_SECRET` | — | **Required.** Min 32 characters |
| `JWT_EXPIRY_HOURS` | `24` | Token expiry duration |
| `STRIPE_SECRET_KEY` | — | Optional. Stripe API key |
| `STRIPE_WEBHOOK_SECRET` | — | Optional. Stripe webhook signing secret |
| `FRONTEND_URL` | `http://localhost:5173` | CORS allowed origin |

---

## Project Structure

```
gym-app/
├── backend/
│   ├── cmd/api/main.go              # Server entry point
│   ├── internal/
│   │   ├── config/                  # Env variable loading
│   │   ├── database/                # PostgreSQL + Redis clients
│   │   ├── handlers/                # HTTP route handlers
│   │   ├── middleware/              # JWT auth + rate limiting
│   │   └── models/                  # Go structs
│   ├── migrations/                  # SQL migration files
│   └── Dockerfile
├── frontend-admin/
│   ├── src/
│   │   ├── components/              # CheckinFeed, StatsCard, QRScanner
│   │   ├── pages/                   # Dashboard, Members, Login
│   │   ├── services/api.ts          # Axios API client
│   │   └── App.tsx
│   └── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```
