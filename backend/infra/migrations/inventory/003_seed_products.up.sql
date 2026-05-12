-- Inserts sample products with stock levels.
--
-- Why seed data?
-- Without it, GET /products returns an empty array — not impressive in a demo.
-- Using real Shoprite-style products (oil, rice, milk, bread, eggs) makes
-- the service feel immediately relevant when demoing to ShopriteX.
--
-- These are products a South African retailer would actually carry.

INSERT INTO products (id, name, description, price, sku, category) VALUES
    (
        'a1b2c3d4-0001-0001-0001-000000000001',
        'Sunflower Oil 2L',
        'Pure sunflower cooking oil, ideal for frying and baking',
        89.99,
        'OIL-SF-2L',
        'Groceries'
    ),
    (
        'a1b2c3d4-0002-0002-0002-000000000002',
        'Basmati Rice 5kg',
        'Premium long grain basmati rice',
        129.99,
        'RICE-BAS-5KG',
        'Groceries'
    ),
    (
        'a1b2c3d4-0003-0003-0003-000000000003',
        'Full Cream Milk 1L',
        'Fresh full cream pasteurised milk',
        24.99,
        'DAIRY-MLK-FC-1L',
        'Dairy'
    ),
    (
        'a1b2c3d4-0004-0004-0004-000000000004',
        'White Bread 700g',
        'Soft sliced white bread',
        17.99,
        'BREAD-WH-700G',
        'Bakery'
    ),
    (
        'a1b2c3d4-0005-0005-0005-000000000005',
        'Free Range Eggs 18 Pack',
        'Fresh free range large eggs',
        59.99,
        'EGGS-FR-18PK',
        'Dairy'
    );

-- Insert corresponding stock levels for each product.
-- quantity = physical units in warehouse
-- reserved = 0 (no pending orders yet)
INSERT INTO stock_levels (product_id, quantity, reserved, warehouse_id) VALUES
    ('a1b2c3d4-0001-0001-0001-000000000001', 150, 0, 'main'),
    ('a1b2c3d4-0002-0002-0002-000000000002', 200, 0, 'main'),
    ('a1b2c3d4-0003-0003-0003-000000000003', 500, 0, 'main'),
    ('a1b2c3d4-0004-0004-0004-000000000004', 300, 0, 'main'),
    ('a1b2c3d4-0005-0005-0005-000000000005', 250, 0, 'main');