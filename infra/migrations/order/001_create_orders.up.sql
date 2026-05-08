-- 001_create_orders.up.sql
-- Creates the orders table for the Order Service.

CREATE TABLE IF NOT EXISTS orders (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),

    -- user_id is stored as plain UUID — no FK to users table.
    -- Order Service does not own the users table (Auth Service does).
    -- Cross-service FK constraints would couple two databases together.
    -- User identity is validated via JWT, not via database constraint.
    user_id      UUID          NOT NULL,

    -- Order lifecycle: PENDING → PROCESSING → CONFIRMED
    --                                       → FAILED
    --                PENDING → CANCELLED
    status       VARCHAR(50)   NOT NULL DEFAULT 'PENDING',

    -- Total calculated as sum(quantity × unit_price) for all items.
    -- Stored here for quick retrieval without joining order_items.
    total_amount NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),

    -- Optional customer notes
    notes        TEXT,

    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── Constraints ───────────────────────────────────────────────────────────────
ALTER TABLE orders
    ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING', 'PROCESSING', 'CONFIRMED', 'FAILED', 'CANCELLED'));

-- ── Indexes ────────────────────────────────────────────────────────────────
-- Most queries filter by user_id — GET /orders lists by authenticated user
CREATE INDEX IF NOT EXISTS idx_orders_user_id
    ON orders(user_id);

-- Status filtering — admin queries, background jobs
CREATE INDEX IF NOT EXISTS idx_orders_status
    ON orders(status);

-- List orders newest first
CREATE INDEX IF NOT EXISTS idx_orders_created
    ON orders(created_at DESC);

-- Composite — user's orders sorted by date (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_orders_user_created
    ON orders(user_id, created_at DESC);

-- Idempotency — unique per user, allows different users to use the same key
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idempotency_key
    ON orders(user_id, idempotency_key);

-- ── Auto-update updated_at ─────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_orders_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_orders_updated_at();
