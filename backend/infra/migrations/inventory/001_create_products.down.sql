-- 001_create_products.down.sql
-- Rollback: drops the products table and its trigger/function.
-- Must drop in reverse order of creation.

DROP TRIGGER IF EXISTS products_updated_at ON products;
DROP FUNCTION IF EXISTS update_products_updated_at();
DROP TABLE IF EXISTS products;