-- ============================================================
-- 000001_init_schema.down.sql
-- Rollback all schema changes from 000001_init_schema.up.sql
-- ============================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS set_updated_at_subscriptions ON subscriptions;
DROP TRIGGER IF EXISTS set_updated_at_plans         ON membership_plans;
DROP TRIGGER IF EXISTS set_updated_at_users         ON users;

DROP FUNCTION IF EXISTS trigger_set_updated_at();

-- Drop tables in reverse FK dependency order
DROP TABLE IF EXISTS check_ins;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS membership_plans;
DROP TABLE IF EXISTS users;

-- Drop enum types
DROP TYPE IF EXISTS plan_interval;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS user_role;
