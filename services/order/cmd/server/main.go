// Package main is the entry point for the Order Service.
// Composition root — wires all dependencies together explicitly.
package main

import (
	"context"
	"time"

	"retail-platform/order/internal/client"
	"retail-platform/order/internal/config"
	"retail-platform/order/internal/database"
	"retail-platform/order/internal/handler"
	"retail-platform/order/internal/repository"
	"retail-platform/order/internal/server"
	"retail-platform/order/internal/service"
	"retail-platform/order/internal/worker"
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
	log := logger.New("order-service")
	log.Info().Str("env", cfg.AppEnv).Msg("starting order service")

	// Step 3: Database connection pool
	ctx := context.Background()
	db, err := database.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Msg("database connection pool established")

	// Step 4: Service token manager — logs in with service account immediately
	// Order Service needs a JWT to call Inventory Service
	tokenManager, err := client.NewServiceTokenManager(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to authenticate service account")
	}
	log.Info().Msg("service account authenticated")

	// Step 5: Inventory client — uses token manager on every request
	inventoryClient := client.NewInventoryClient(cfg, tokenManager)

	// Step 6: Repository
	orderRepo := repository.NewPostgresOrderRepo(db)

	// Step 7: Event bus — order → notification
	eventBus := events.NewBus()

	// Step 8: Worker processor + pool
	processor := worker.NewOrderProcessor(orderRepo, inventoryClient, eventBus, log)
	pool := worker.NewWorkerPool(cfg.WorkerPoolSize, processor, log)

	// Step 9: Start workers BEFORE HTTP server accepts requests
	pool.Start(ctx)

	// Step 10: JWT manager — validates tokens issued by Auth Service
	jwtManager := jwt.NewManager(cfg.JWTSecret, 15*time.Minute, 168*time.Hour)

	// Step 11: Validator
	v := validator.New()

	// Step 12: Service
	orderService := service.NewOrderService(orderRepo, inventoryClient, pool, eventBus, log)

	// Step 13: Handler
	orderHandler := handler.NewOrderHandler(orderService, v, log)

	// Step 14: Router + Server
	router := server.NewRouter(orderHandler, jwtManager)
	srv := server.New(router, db, pool, cfg, log)

	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
