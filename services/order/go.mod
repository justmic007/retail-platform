module retail-platform/order

go 1.25.0

require (
	github.com/caarlos0/env/v10 v10.0.0
	github.com/gin-gonic/gin v1.12.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/joho/godotenv v1.5.1
	retail-platform/pkg v0.0.0
)

replace retail-platform/pkg => ../../pkg
