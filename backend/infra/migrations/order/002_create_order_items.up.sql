-- 002_create_order_items.up.sql
-- Creates the order_items table.
--
-- Key design: product_name and unit_price are SNAPSHOTS.
-- They are copied from the request at order creation time.
-- If Sunflower Oil price changes from R89.99 to R99.99 tomorrow,
-- existing orders still correctly show R89.99.
--
-- Never JOIN to the products table for historical order data.
-- The snapshot IS the historical record.

CREATE TABLE IF NOT EXISTS order_items (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),

    -- ON DELETE CASCADE: if an order is deleted, its items are deleted too
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,

    -- product_id stored for reference — but we never join to products table
    -- to get name or price (those are snapshots below)
    product_id   UUID          NOT NULL,

    -- Snapshot of product name at order time
    product_name VARCHAR(255)  NOT NULL,

    quantity     INTEGER       NOT NULL CHECK (quantity > 0),

    -- Snapshot of price at order time — NUMERIC not FLOAT (exact decimal)
    unit_price   NUMERIC(10,2) NOT NULL CHECK (unit_price >= 0),

    -- Denormalised total = quantity * unit_price
    -- Stored to avoid recalculation on every read
    total_price  NUMERIC(10,2) NOT NULL CHECK (total_price >= 0),

    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── Indexes ────────────────────────────────────────────────────────────────
-- Primary access pattern: get all items for an order
CREATE INDEX IF NOT EXISTS idx_order_items_order_id
    ON order_items(order_id);

-- Secondary: find all orders containing a specific product
-- (useful for analytics — "how many orders included Sunflower Oil?")
CREATE INDEX IF NOT EXISTS idx_order_items_product_id
    ON order_items(product_id);