// Package server provides the HTTP server for the Notification Service.
// Minimal HTTP — only /health and /ready. No Gin needed — standard net/http.
// The Notification Service is event-driven, not request-driven.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"retail-platform/notification/internal/config"
	"retail-platform/pkg/logger"
)

// Server wraps the HTTP server.
type Server struct {
	httpServer *http.Server
	log        *logger.Logger
}

// New creates a new Server.
func New(cfg *config.Config, log *logger.Logger) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"notification","status":"ok"}`))
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"notification","status":"ready"}`))
	})

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		log: log,
	}
}

// Run starts the HTTP server and blocks until shutdown signal.
func (s *Server) Run() error {
	serverErr := make(chan error, 1)
	go func() {
		s.log.Info().Str("addr", s.httpServer.Addr).Msg("notification service starting")
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

	s.log.Info().Msg("notification service stopped cleanly")
	return nil
}
