// Package config loads all environment variables for the Inventory Service.
package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the Inventory Service.
type Config struct {
	AppEnv string `env:"APP_ENV" envDefault:"development"`
	Port   int    `env:"INVENTORY_PORT" envDefault:"8082"`

	// Database
	DatabaseURL string `env:"DATABASE_URL,required"`

	// Redis — for stock level caching
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`

	// JWT — must match Auth Service secret exactly
	// Used to validate tokens on every protected request
	JWTSecret string `env:"JWT_SECRET,required"`

	// Cache TTL — how long stock levels stay cached in Redis
	// After this duration, the next request hits Postgres and re-caches
	CacheTTL time.Duration `env:"CACHE_TTL" envDefault:"5m"`

	// Low stock threshold — publish StockLow event when stock drops below this
	// Notification Service listens for this event
	LowStockThreshold int `env:"LOW_STOCK_THRESHOLD" envDefault:"10"`
}

// Load reads all environment variables and returns a populated Config.
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}
