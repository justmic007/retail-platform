-- Creates the stock_levels table.
--
-- Two key fields: quantity and reserved.
--
-- quantity = total physical units in the warehouse
-- reserved = units locked by pending/processing orders
-- available = quantity - reserved (what customers can actually buy)
--
-- Example:
--   quantity = 10  (10 units physically in warehouse)
--   reserved = 3   (3 units locked by pending orders)
--   available = 7  (7 units available to new customers)
--
-- When order placed:   reserved  += N   (lock the units)
-- When order shipped:  quantity  -= N   (consume from stock)
--                      reserved  -= N   (release the lock)
-- When order cancelled: reserved -= N   (release the lock, nothing consumed)

CREATE TABLE IF NOT EXISTS stock_levels (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    -- FK to products — ON DELETE CASCADE means if a product is deleted,
    -- its stock record is automatically deleted too.
    product_id   UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,

    -- Total physical units. CHECK prevents negative stock at DB level.
    -- This is defence-in-depth — even if application code has a bug,
    -- the database refuses to store negative stock.
    quantity     INTEGER      NOT NULL DEFAULT 0 CHECK (quantity >= 0),

    -- Units reserved by pending orders.
    reserved     INTEGER      NOT NULL DEFAULT 0 CHECK (reserved >= 0),

    -- warehouse_id enables multi-warehouse support in the future.
    -- For now, everything goes to 'main'.
    warehouse_id VARCHAR(100) NOT NULL DEFAULT 'main',

    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- One stock record per product per warehouse.
    -- Prevents duplicate stock records for the same product.
    UNIQUE(product_id, warehouse_id)
);

-- ── Indexes ────────────────────────────────────────────────────────────────
-- Most stock queries look up by product_id
CREATE INDEX IF NOT EXISTS idx_stock_product_id ON stock_levels(product_id);

-- ── Auto-update updated_at ─────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_stock_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_levels_updated_at
    BEFORE UPDATE ON stock_levels
    FOR EACH ROW
    EXECUTE FUNCTION update_stock_updated_at();