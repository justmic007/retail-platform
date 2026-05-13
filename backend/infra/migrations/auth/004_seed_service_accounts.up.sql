-- 004_seed_service_accounts.up.sql
-- Creates a service account for Order Service in auth_db.
-- email_verified is set to TRUE in migration 005 after the column is added.

INSERT INTO users (email, password_hash, role)
VALUES (
    'order-service@internal.retailplatform.com',
    '$2a$12$1GzgUdKSmtqLZsYqOyc3QeG/k0lMOtIhAOCNkJaxRcFlBDGV.ae5S',
    'customer'
)
ON CONFLICT (email) DO NOTHING;
