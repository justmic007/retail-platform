-- 002_create_refresh_tokens.up.sql
-- Migration: Create the refresh_tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
);

-- ── Indexes ────────────────────────────────────────────────────────────────
 
-- Index on token: every refresh request looks up the token by value.

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);

-- Index on user_id: used when logging out all devices (delete all user's tokens).
-- Also used to count how many active sessions a user has.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Index on expires_at: used for a cleanup job that deletes expired tokens.
-- Without this, the cleanup job scans the entire table every time.

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);