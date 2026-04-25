// Package main is the entry point for the Auth Service.
// This file is the COMPOSITION ROOT — the single place where all
// dependencies are created and wired together.
package main

import (
	"context"

	"retail-platform/auth/internal/config"
	"retail-platform/auth/internal/database"
	"retail-platform/auth/internal/handler"
	"retail-platform/auth/internal/repository"
	"retail-platform/auth/internal/server"
	"retail-platform/auth/internal/service"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/logger"
	"retail-platform/pkg/validator"
)

func main() {
	// Step 1: Load config — exits if required env vars are missing
	cfg := config.Load()

	// Step 2: Structured logger
	log := logger.New("auth-service")
	log.Info().Str("env", cfg.AppEnv).Msg("starting auth service")

	// Step 3: Database connection pool — pings DB, fails fast if unreachable
	ctx := context.Background()
	db, err := database.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Msg("database connection pool established")

	// Step 4: JWT Manager — signing secret + TTL config
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// Step 5: Repositories — return interfaces, not concrete types
	userRepo := repository.NewPostgresUserRepo(db)
	tokenRepo := repository.NewPostgresTokenRepo(db)

	// Step 6: Validator
	v := validator.New()

	// Step 7: Service — receives all dependencies via constructor
	authService := service.NewAuthService(userRepo, tokenRepo, jwtManager, cfg, log)

	// Step 8: Handler — thin HTTP layer
	authHandler := handler.NewAuthHandler(authService, v, log)

	// Step 9: Router — registers routes + middleware chain
	router := server.NewRouter(authHandler, jwtManager)

	// Step 10: Server — starts HTTP server, blocks until SIGTERM/SIGINT
	srv := server.New(router, db, cfg, log)
	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
