-- Remove seed data.
-- stock_levels deleted automatically via ON DELETE CASCADE on products.
DELETE FROM products WHERE sku IN (
    'OIL-SF-2L',
    'RICE-BAS-5KG',
    'DAIRY-MLK-FC-1L',
    'BREAD-WH-700G',
    'EGGS-FR-18PK'
);