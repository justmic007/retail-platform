-- Creates the default admin user.
-- Password: Admin@123456 (bcrypt hash below)
-- IMPORTANT: Change this password immediately after first login in production.
--
-- To generate a new bcrypt hash for a different password:
-- go run -e 'import "golang.org/x/crypto/bcrypt"; fmt.Println(string(bcrypt.GenerateFromPassword([]byte("yourpassword"), 12)))'
-- Or use: https://bcrypt-generator.com (cost 12)

INSERT INTO users (email, password_hash, role)
VALUES (
    'admin@retailplatform.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/oK7rkJQyS',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
-- ON CONFLICT means: if admin already exists, skip silently (safe to re-run)