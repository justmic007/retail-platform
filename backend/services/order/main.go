// Package main is the entry point for the Order Service.
package main

import (
	"fmt"
	"os"

	"retail-platform/pkg/logger"
)

func main() {
	log := logger.New("order-service")
	log.Info().Msg("Order Service starting...")
	log.Info().Str("status", "scaffold only").Msg("Day 4 will wire real logic")
	fmt.Println("Order service scaffold OK")
	os.Exit(0)
}
