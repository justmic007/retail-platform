// Package main is the entry point for the Notification Service.
package main

import (
	"fmt"
	"os"

	"retail-platform/pkg/logger"
)

func main() {
	log := logger.New("notification-service")
	log.Info().Msg("Notification Service starting...")
	log.Info().Str("status", "scaffold only").Msg("Day 5 will wire real logic")
	fmt.Println("Notification service scaffold OK")
	os.Exit(0)
}