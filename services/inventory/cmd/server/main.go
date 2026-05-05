// Package main is the entry point for the Inventory Service.
// Composition root — wires all dependencies together explicitly.
package main

import (
	"context"
	"time"

	"retail-platform/inventory/internal/cache"
	"retail-platform/inventory/internal/config"
	"retail-platform/inventory/internal/database"
	"retail-platform/inventory/internal/handler"
	"retail-platform/inventory/internal/repository"
	"retail-platform/inventory/internal/server"
	"retail-platform/inventory/internal/service"
	"retail-platform/pkg/events"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/logger"
	"retail-platform/pkg/validator"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (local development only)
	_ = godotenv.Load()

	// Step 1: Config
	cfg := config.Load()

	// Step 2: Logger
	log := logger.New("inventory-service")
	log.Info().Str("env", cfg.AppEnv).Msg("starting inventory service")

	// Step 3: Database connection pool
	ctx := context.Background()
	db, err := database.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Msg("database connection pool established")

	// Step 4: Redis cache
	stockCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	log.Info().Msg("redis cache connected")

	// Step 5: Event bus — for publishing StockLow events
	eventBus := events.NewBus()

	// Step 6: JWT Manager — validates tokens issued by Auth Service
	jwtManager := jwt.NewManager(cfg.JWTSecret, 15*time.Minute, 168*time.Hour)

	// Step 7: Repositories
	productRepo := repository.NewPostgresProductRepo(db)
	stockRepo := repository.NewPostgresStockRepo(db)

	// Step 8: Validator
	v := validator.New()

	// Step 9: Service
	inventoryService := service.NewInventoryService(
		productRepo, stockRepo, stockCache, eventBus, cfg, log,
	)

	// Step 10: Handler
	inventoryHandler := handler.NewInventoryHandler(inventoryService, v, log)

	// Step 11: Router
	router := server.NewRouter(inventoryHandler, jwtManager)

	// Step 12: Server — blocks until SIGTERM/SIGINT
	srv := server.New(router, db, cfg, log)
	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
