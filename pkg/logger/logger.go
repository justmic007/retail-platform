// Package logger provides a structured logger for all services.
// It wraps zerolog and adds consistent fields across the platform.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger is our custom logger type.
// We wrap zerolog.Logger so we can add methods to it later.
type Logger struct {
	zerolog.Logger
}

// New creates a new logger for a given service.
// In production (LOG_FORMAT=json), it writes JSON to stdout.
// In development, it writes pretty-printed colored output.
//
// Usage:
//
//	log := logger.New("auth-service")
//	log.Info().Str("user_id", "123").Msg("user logged in")
func New(serviceName string) *Logger {
	// Determine output format from environment
	// JSON in production — Loki/Grafana can parse it
	// Pretty in development — humans can read it
	var zl zerolog.Logger

	if os.Getenv("LOG_FORMAT") == "json" {
		zl = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Str("service", serviceName). // Every log line knows which service it came from
			Logger()
	} else {
		// Pretty output for local development
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		zl = zerolog.New(output).
			With().
			Timestamp().
			Str("service", serviceName).
			Logger()
	}

	return &Logger{zl}
}

// WithRequestID returns a new logger with the request ID attached.
// This lets you trace a single request across all log lines.
//
// Usage:
//
//	log := logger.WithRequestID("req-abc-123")
//	log.Info().Msg("processing order") // every line will have request_id
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{l.With().Str("request_id", requestID).Logger()}
}
