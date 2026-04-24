// Package config loads all environment variables for the Auth Service
// into a single typed struct. This is the ONLY place in the entire service
// that reads from the environment.
package config

import (
	"log"
	"time"

	// caarlos0/env reads struct field tags and automatically parses
	// environment variables into the correct Go types.
	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the Auth Service.
// Each field maps to an environment variable via the `env` struct tag.
//
// Struct tags explained:
//
//	env:"VAR_NAME"          → reads from this environment variable
//	envDefault:"value"      → uses this value if the variable is not set
//	env:"VAR_NAME,required" → fails to start if this variable is missing
//
// Using 'required' for secrets means the service refuses to start
// if JWT_SECRET is not set — better than running with an empty secret.
type Config struct {
	AppEnv          string        `env:"APP_ENV" envDefault:"development"`
	Port            int           `env:"AUTH_PORT" envDefault:"8080"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	JWTSecret       string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" envDefault:"168h"`

	// ── Rate Limiting ────────────────────────────────────────────────────────
	// Limits login attempts to prevent brute-force attacks.
	RateLimitRequests int    `env:"RATE_LIMIT_REQUESTS" envDefault:"10"`
	RateLimitPeriod   string `env:"RATE_LIMIT_PERIOD" envDefault:"1m"`

	// ── Bcrypt ──────────────────────────────────────────────────────────────
	// Too slow for brute force, fast enough for user login.
	BcryptCost int `env:"BCRYPT_COST" envDefault:"12"`
}

// Load reads all environment variables and returns a populated Config.
// It is called exactly once in main.go before anything else.
// If any required variable is missing, Load logs the error and exits
// the process immediately — fail fast, fail loud.
func Load() *Config {
	cfg := &Config{}

	// env.Parse reads all env tags on the struct and populates the fields.
	// It returns an error if required fields are missing or values can't
	// be parsed into the expected types (e.g. "abc" for an int field).
	if err := env.Parse(cfg); err != nil {
		// log.Fatalf prints the message and calls os.Exit(1).
		// We use it here because without valid config, the service
		// cannot function at all — there is no point continuing.
		log.Fatalf("failed to load config: %v", err)
	}

	return cfg
}
