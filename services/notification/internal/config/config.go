// Package config loads all environment variables for the Notification Service.
package config

import (
	"log"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the Notification Service.
// No database — stateless by design.
type Config struct {
	Port      int    `env:"NOTIFICATION_PORT" envDefault:"8083"`
	AppEnv    string `env:"APP_ENV" envDefault:"development"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"pretty"`

	// Redis — for event bus (Redis Pub/Sub)
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`

	// WarehouseEmail — recipient for internal low stock alerts
	WarehouseEmail string `env:"WAREHOUSE_EMAIL" envDefault:"workfromhomenoni@gmail.com"`

	// Brevo — transactional email provider
	BrevoAPIKey    string `env:"BREVO_SEND_EMAIL_API_KEY"`
	EmailFrom      string `env:"EMAIL_FROM" envDefault:"noreply@retailplatform.com"`
	EmailFromName  string `env:"EMAIL_FROM_NAME" envDefault:"Retail Platform"`
}

// Load reads all environment variables and returns a populated Config.
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}
