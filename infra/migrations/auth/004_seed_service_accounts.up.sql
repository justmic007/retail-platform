-- 003_seed_service_accounts.up.sql
-- Creates a service account for Order Service in auth_db.
--
-- Order Service uses this account to authenticate with Inventory Service.
-- It logs in at startup, gets a JWT, and passes it on every call to
-- Inventory Service — same auth flow as a regular user.
--
-- Password: OrderService@123456 (bcrypt hash below, cost 12)
-- Store the real password in ORDER_SERVICE_PASSWORD env var.
-- NEVER commit the plain text password.

INSERT INTO users (email, password_hash, role, email_verified)
VALUES (
    'order-service@internal.retailplatform.com',
    '$2a$12$1GzgUdKSmtqLZsYqOyc3QeG/k0lMOtIhAOCNkJaxRcFlBDGV.ae5S',
    'customer',
    TRUE
)
ON CONFLICT (email) DO NOTHING;
