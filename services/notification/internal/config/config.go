// Package config loads all environment variables for the Notification Service.
package config

import (
	"log"

	"github.com/caarlos0/env/v10"
)

// This Config holds all configuration for the Notification Service.
// No database, no Redis.
type Config struct {
	Port      int    `env:"NOTIFICATION_PORT" envDefault:"8083"`
	AppEnv    string `env:"APP_ENV" envDefault:"development"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"pretty"`
}

// Load reads all environment variables and returns a populated Config.
func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}
