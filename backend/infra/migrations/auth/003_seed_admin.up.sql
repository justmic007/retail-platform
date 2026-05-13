-- Creates the default admin user.
-- Password: Admin@123456 (bcrypt hash below)
-- email_verified is set to TRUE in migration 005 after the column is added.

INSERT INTO users (email, password_hash, role)
VALUES (
    'admin@retailplatform.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/oK7rkJQyS',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
