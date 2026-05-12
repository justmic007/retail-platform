// Package config loads all environment variables for the Auth Service
// into a single typed struct. This is the ONLY place in the entire service
// that reads from the environment.
package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the Auth Service.
type Config struct {
	AppEnv          string        `env:"APP_ENV" envDefault:"development"`
	Port            int           `env:"AUTH_PORT" envDefault:"8080"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	JWTSecret       string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" envDefault:"168h"`

	RateLimitRequests int    `env:"RATE_LIMIT_REQUESTS" envDefault:"10"`
	RateLimitPeriod   string `env:"RATE_LIMIT_PERIOD" envDefault:"1m"`

	BcryptCost int `env:"BCRYPT_COST" envDefault:"12"`

	// Brevo — transactional email for verification emails
	BrevoAPIKey   string `env:"BREVO_SEND_EMAIL_API_KEY"`
	EmailFrom     string `env:"EMAIL_FROM" envDefault:"noreply@retailplatform.com"`
	EmailFromName string `env:"EMAIL_FROM_NAME" envDefault:"Retail Platform"`

	// AppBaseURL is used to build the verification link sent in emails
	AppBaseURL string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
}

// Load reads all environment variables and returns a populated Config.
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}
