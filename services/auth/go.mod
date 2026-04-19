module retail-platform/auth

go 1.26

require (
    // Shared packages from our own monorepo (resolved via go.work locally)
    retail-platform/pkg v0.0.0

    // Web framework — like Express but for Go
    github.com/gin-gonic/gin v1.9.1

    // Postgres driver — the best one for Go
    github.com/jackc/pgx/v5 v5.5.5

    // Database migrations
    github.com/golang-migrate/migrate/v4 v4.17.0

    // Password hashing
    golang.org/x/crypto v0.21.0

    // Environment variable loading
    github.com/joho/godotenv v1.5.1

    // Config management
    github.com/caarlos0/env/v10 v10.0.0

    // Rate limiting
    github.com/ulule/limiter/v3 v3.11.2
)

// This replace directive tells Go: when you see retail-platform/pkg,
// find it at ../../../pkg (relative to this go.mod).
// go.work handles this automatically — this is just for clarity.
replace retail-platform/pkg => ../../pkg