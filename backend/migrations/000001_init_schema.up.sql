-- ============================================================
-- 000001_init_schema.up.sql
-- Initial schema for the Gym Membership Application
-- ============================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ENUM TYPES
-- ============================================================

CREATE TYPE user_role AS ENUM ('admin', 'trainer', 'member');

CREATE TYPE subscription_status AS ENUM (
    'active',
    'past_due',
    'canceled',
    'paused',
    'trialing'
);

CREATE TYPE plan_interval AS ENUM ('monthly', 'yearly');

-- ============================================================
-- USERS
-- ============================================================

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    role            user_role NOT NULL DEFAULT 'member',
    phone           VARCHAR(30),
    avatar_url      TEXT,
    -- Unique QR code token assigned to each member
    qr_token        UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email     ON users (email);
CREATE INDEX idx_users_qr_token  ON users (qr_token);
CREATE INDEX idx_users_role      ON users (role);

-- ============================================================
-- MEMBERSHIP PLANS
-- ============================================================

CREATE TABLE membership_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    price_cents         INTEGER NOT NULL,          -- stored in smallest currency unit
    currency            CHAR(3) NOT NULL DEFAULT 'USD',
    billing_interval    plan_interval NOT NULL DEFAULT 'monthly',
    stripe_price_id     VARCHAR(255),              -- Stripe Price ID for this plan
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plans_stripe_price_id ON membership_plans (stripe_price_id);

-- Seed default plans
INSERT INTO membership_plans (name, description, price_cents, billing_interval)
VALUES
    ('Monthly Basic',  'Full gym access — billed monthly',   2999, 'monthly'),
    ('Yearly Pro',     'Full gym access — billed yearly',   29900, 'yearly');

-- ============================================================
-- SUBSCRIPTIONS
-- ============================================================

CREATE TABLE subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id                 UUID NOT NULL REFERENCES membership_plans(id),
    status                  subscription_status NOT NULL DEFAULT 'active',
    stripe_customer_id      VARCHAR(255),
    stripe_subscription_id  VARCHAR(255) UNIQUE,
    current_period_start    TIMESTAMPTZ,
    current_period_end      TIMESTAMPTZ,
    canceled_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id               ON subscriptions (user_id);
CREATE INDEX idx_subscriptions_status                ON subscriptions (status);
CREATE INDEX idx_subscriptions_stripe_subscription_id ON subscriptions (stripe_subscription_id);
CREATE INDEX idx_subscriptions_stripe_customer_id    ON subscriptions (stripe_customer_id);

-- ============================================================
-- CHECK-INS (ACCESS LOG)
-- ============================================================

CREATE TABLE check_ins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    qr_token        UUID NOT NULL,
    access_granted  BOOLEAN NOT NULL,
    denial_reason   TEXT,                          -- populated when access_granted = false
    checked_in_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address      INET,
    notes           TEXT                           -- manual override notes by front-desk staff
);

CREATE INDEX idx_check_ins_user_id      ON check_ins (user_id);
CREATE INDEX idx_check_ins_checked_in_at ON check_ins (checked_in_at DESC);
CREATE INDEX idx_check_ins_access_granted ON check_ins (access_granted);

-- ============================================================
-- UPDATED_AT TRIGGER (auto-update timestamp)
-- ============================================================

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_users
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_plans
    BEFORE UPDATE ON membership_plans
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_subscriptions
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
