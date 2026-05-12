// server.go — HTTP server lifecycle management.
// Starts the server, handles graceful shutdown on SIGTERM/SIGINT.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"retail-platform/auth/internal/config"
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

// New creates a new Server with configured timeouts.
//
// Timeout explanation:
//   - ReadTimeout: max time to read the complete request (headers + body).
//     Prevents Slowloris attacks where a client sends headers very slowly.
//   - WriteTimeout: max time to write the complete response.
//     Prevents handlers from hanging forever on slow clients.
//   - IdleTimeout: max time to wait for the next request on a keep-alive connection.
//     Frees up connections from idle clients.
func New(router *gin.Engine, db *pgxpool.Pool, cfg *config.Config, log *logger.Logger) *Server {
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		db:         db,
		log:        log,
	}
}

// Run starts the HTTP server and blocks until a shutdown signal is received.
//
// Graceful shutdown sequence:
//  1. Start HTTP server in a goroutine (so it doesn't block)
//  2. Block main goroutine waiting for OS signal (SIGTERM or SIGINT)
//  3. Signal received → stop accepting new requests immediately
//  4. Wait up to 30 seconds for in-flight requests to complete
//  5. Close the database connection pool
//  6. Return — main() exits cleanly
//
// Why does graceful shutdown matter?
// Kubernetes sends SIGTERM when rolling out a new deployment.
// Without graceful shutdown: pod dies mid-request → user gets 502 error.
// With graceful shutdown: pod finishes current requests → no 502 errors.
func (s *Server) Run() error {
	// ── Start server in background goroutine ──────────────────────────────
	// ListenAndServe blocks — if we called it directly, nothing after it
	// would run. By wrapping it in a goroutine, main continues to the
	// signal-waiting code below.
	//
	// This is your first real goroutine usage:
	//   go func() { ... }()  creates a new goroutine and runs the function
	//   concurrently with the current goroutine.
	serverErr := make(chan error, 1)
	go func() {
		s.log.Info().Str("addr", s.httpServer.Addr).Msg("auth service starting")

		// ListenAndServe returns http.ErrServerClosed when Shutdown() is called.
		// That's expected — not a real error. We filter it out below.
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
	}()

	// ── Wait for shutdown signal ──────────────────────────────────────────
	// signal.NotifyContext creates a context that is cancelled when
	// SIGINT (Ctrl+C) or SIGTERM (Kubernetes pod termination) is received.
	//
	// We block here — main goroutine waits until the context is cancelled.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Block until either:
	//   - A shutdown signal arrives (ctx.Done() fires)
	//   - The server crashes (serverErr receives an error)
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		s.log.Info().Msg("shutdown signal received")
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────
	// Give in-flight requests up to 30 seconds to complete.
	// After 30 seconds, any remaining requests are forcefully terminated.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown stops accepting new connections and waits for active ones.
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.log.Error().Err(err).Msg("server shutdown error")
		return fmt.Errorf("server shutdown: %w", err)
	}

	// Close the database connection pool after all HTTP connections are done.
	// Order matters: close HTTP first, then DB.
	// If you close DB first, in-flight requests that need the DB will fail.
	s.db.Close()
	s.log.Info().Msg("auth service stopped cleanly")

	return nil
}
