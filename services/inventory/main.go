// Package main is the entry point for the Inventory Service.
package main

import (
	"fmt"
	"os"

	"retail-platform/pkg/logger"
)

func main() {
	log := logger.New("inventory-service")
	log.Info().Msg("Inventory Service starting...")
	log.Info().Str("status", "scaffold only").Msg("Day 3 will wire real logic")
	fmt.Println("Inventory service scaffold OK")
	os.Exit(0)
}
