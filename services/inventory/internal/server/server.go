// Package server — HTTP server lifecycle for Inventory Service.
// Identical graceful shutdown pattern as Auth Service.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"retail-platform/inventory/internal/config"
	"retail-platform/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	httpServer *http.Server
	db         *pgxpool.Pool
	log        *logger.Logger
}

// New creates a new Server.
func New(router *gin.Engine, db *pgxpool.Pool, cfg *config.Config, log *logger.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		db:  db,
		log: log,
	}
}

// Run starts the HTTP server and blocks until shutdown signal.
func (s *Server) Run() error {
	serverErr := make(chan error, 1)
	go func() {
		s.log.Info().Str("addr", s.httpServer.Addr).Msg("inventory service starting")
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		s.log.Info().Msg("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.db.Close()
	s.log.Info().Msg("inventory service stopped cleanly")
	return nil
}
