module retail-platform/inventory

go 1.26

require (
    retail-platform/pkg v0.0.0
    github.com/gin-gonic/gin v1.9.1
    github.com/jackc/pgx/v5 v5.5.5
    github.com/golang-migrate/migrate/v4 v4.17.0
    github.com/redis/go-redis/v9 v9.5.1
    github.com/joho/godotenv v1.5.1
    github.com/caarlos0/env/v10 v10.0.0
)

replace retail-platform/pkg => ../../pkg