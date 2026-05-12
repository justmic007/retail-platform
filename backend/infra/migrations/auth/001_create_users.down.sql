-- Rollback: Drop the users table in reverse order what the up did
DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
-- DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;