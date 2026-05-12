// Package config loads all the environment variables for the Order Service
// into a single typed struct. This is the ONLY place in the entire service
// that reads from the environment.
package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the Order Service.
type Config struct {
	AppEnv      string `env:"APP_ENV" envDefault:"development"`
	Port        int    `env:"ORDER_PORT" envDefault:"8081"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	LogFormat   string `env:"LOG_FORMAT" envDefault:"pretty"`

	// ── JWT ────────────────────────────────────────────────────────────────
	JWTSecret string `env:"JWT_SECRET,required"`

	// Inventory Service - where Order Service calls to check stock and reserve items.
	InventoryServiceURL    string        `env:"INVENTORY_SERVICE_URL,required"`
	InventoryClientTimeout time.Duration `env:"INVENTORY_CLIENT_TIMEOUT" envDefault:"5s"`

	// Auth Service - used by ServiceTokenManager to get a JWT at startup for inter-service auth.
	AuthServiceURL       string `env:"AUTH_SERVICE_URL,required"`
	OrderServiceEmail    string `env:"ORDER_SERVICE_EMAIL,required"`
	OrderServicePassword string `env:"ORDER_SERVICE_PASSWORD,required"`

	// Worker pool — how many goroutines process orders concurrently
	WorkerPoolSize int `env:"WORKER_POOL_SIZE" envDefault:"10"`

	// Redis — for event bus (Redis Pub/Sub)
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
}

// Load reads all environment variables and returns a populated Config.
// Exits immediately if any required variable is missing or cannot be parsed.
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}
