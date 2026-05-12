// Package main is the entry point for the Auth Service.
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
	"retail-platform/pkg/mailer"
	"retail-platform/pkg/validator"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	log := logger.New("auth-service")
	log.Info().Str("env", cfg.AppEnv).Msg("starting auth service")

	ctx := context.Background()
	db, err := database.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Msg("database connection pool established")

	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	userRepo := repository.NewPostgresUserRepo(db)
	tokenRepo := repository.NewPostgresTokenRepo(db)
	verificationRepo := repository.NewPostgresVerificationTokenRepo(db)

	m := mailer.New(cfg.BrevoAPIKey, cfg.EmailFrom, cfg.EmailFromName)

	v := validator.New()

	authService := service.NewAuthService(userRepo, tokenRepo, verificationRepo, jwtManager, m, cfg, log)

	authHandler := handler.NewAuthHandler(authService, v, log)

	router := server.NewRouter(authHandler, jwtManager)

	srv := server.New(router, db, cfg, log)
	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
