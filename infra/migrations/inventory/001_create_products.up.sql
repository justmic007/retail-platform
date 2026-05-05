-- 001_create_products.up.sql
-- Creates the products table for the Inventory Service.
--
-- Why separate products from stock_levels?
-- Products change rarely (name, price, description).
-- Stock changes constantly (every order, every delivery).
-- Separating them means high-frequency stock updates never lock
-- the products table. Also enables multi-warehouse support later.

CREATE TABLE IF NOT EXISTS products (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),

    -- SKU = Stock Keeping Unit — the unique product identifier used in retail.
    sku         VARCHAR(100)  NOT NULL UNIQUE,

    name        VARCHAR(255)  NOT NULL,
    description TEXT,

    -- NUMERIC(10,2) = up to 10 digits, 2 decimal places.
    -- Never use FLOAT for money — floating point arithmetic is imprecise.
    -- NUMERIC is exact. 1299.99 stays 1299.99, not 1299.9900000001.
    price       NUMERIC(10,2) NOT NULL CHECK (price >= 0),

    category    VARCHAR(100),

    -- Soft delete — mark inactive instead of deleting.
    -- Deleted products may still be referenced by historical orders.
    is_active   BOOLEAN       NOT NULL DEFAULT true,

    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── Indexes ────────────────────────────────────────────────────────────────
-- SKU lookups — Order Service calls GET /products?sku=OIL-SF-2L
CREATE INDEX IF NOT EXISTS idx_products_sku      ON products(sku);

-- Category filtering — GET /products?category=Groceries
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- Active product filtering — most queries only want active products
CREATE INDEX IF NOT EXISTS idx_products_active   ON products(is_active);

-- ── Auto-update updated_at ─────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_products_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_products_updated_at();