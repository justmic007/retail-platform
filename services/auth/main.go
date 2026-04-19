// Package main is the entry point for the Auth Service.
// TODO: wire together config, database, repository,
// service, handler, and server. 
// TODO:Test it just confirms the service compiles.
package main

import (
	"fmt"
	"os"

	// In Go, imports must be used or the compiler refuses to compile.
	// This is one of the most jarring things coming from Node.js/Python
	// but it keeps codebases clean.
	"retail-platform/pkg/logger"
)

func main() {
	log := logger.New("auth-service")

	log.Info().Msg("Auth Service starting...")
	log.Info().Str("status", "scaffold only").Msg("Day 2 will wire real logic")

	// os.Exit(0) means "exited successfully"
	// We exit here today — on Day 2 we'll start a real HTTP server instead
	fmt.Println("Auth service scaffold OK")
	os.Exit(0)
}