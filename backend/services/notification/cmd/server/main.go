// Package main is the entry point for the Notification Service.
// Composition root — creates the Redis event bus, wires handlers and dispatcher,
// starts the dispatcher goroutine, then starts the HTTP server.
package main

import (
	"context"

	"retail-platform/notification/internal/config"
	"retail-platform/notification/internal/dispatcher"
	"retail-platform/notification/internal/handler"
	"retail-platform/notification/internal/server"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (local development only)
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	// Step 1: Config
	cfg := config.Load()

	// Step 2: Logger
	log := logger.New("notification-service")
	log.Info().Str("env", cfg.AppEnv).Msg("starting notification service")

	// Step 3: Event bus — Redis Pub/Sub for cross-service event delivery
	// Notification Service subscribes to events published by Order and Inventory services
	eventBus, err := events.NewRedisBus(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis event bus")
	}
	log.Info().Msg("redis event bus connected")

	// Step 4: Handlers
	emailHandler := handler.NewEmailHandler(cfg.BrevoAPIKey, cfg.EmailFrom, cfg.EmailFromName, log)
	internalHandler := handler.NewInternalHandler(cfg.BrevoAPIKey, cfg.EmailFrom, cfg.EmailFromName, cfg.WarehouseEmail, log)

	// Step 5: Dispatcher — wires handlers to the event bus
	d := dispatcher.NewDispatcher(eventBus, emailHandler, internalHandler, log)

	// Step 6: Start dispatcher in background goroutine
	// ctx is tied to the server lifecycle — when server shuts down, ctx is cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)
	log.Info().Msg("dispatcher started")

	// Step 7: HTTP server — blocks until SIGTERM/SIGINT
	srv := server.New(cfg, log)
	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
