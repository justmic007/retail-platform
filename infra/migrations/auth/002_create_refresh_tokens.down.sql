-- 002_create_refresh_tokens.down.sql
-- Rollback: Drop the refresh_tokens table
--
-- Drop the table — indexes are dropped automatically with the table.
DROP TABLE IF EXISTS refresh_tokens;