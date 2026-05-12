// Package server — HTTP server lifecycle for Order Service.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"retail-platform/order/internal/config"
	"retail-platform/order/internal/worker"
	"retail-platform/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	httpServer *http.Server
	db         *pgxpool.Pool
	pool       *worker.WorkerPool
	log        *logger.Logger
}

// New creates a new Server.
func New(router *gin.Engine, db *pgxpool.Pool, pool *worker.WorkerPool, cfg *config.Config, log *logger.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		db:   db,
		pool: pool,
		log:  log,
	}
}

// Run starts the HTTP server and blocks until shutdown signal.
// Shutdown sequence: HTTP server → worker pool → database.
// Order matters — stop accepting requests before draining workers,
// drain workers before closing the DB they depend on.
func (s *Server) Run() error {
	serverErr := make(chan error, 1)
	go func() {
		s.log.Info().Str("addr", s.httpServer.Addr).Msg("order service starting")
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

	// 1. Stop accepting new HTTP requests
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	// 2. Drain the worker pool — all in-flight orders complete
	s.pool.Shutdown()

	// 3. Close DB — safe now, no workers are running
	s.db.Close()

	s.log.Info().Msg("order service stopped cleanly")
	return nil
}
