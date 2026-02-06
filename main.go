package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/ruanpelissoli/lootstash-marketplace-api/cmd"
)

func init() {
	// Load .env file before any other init() functions read env vars
	_ = godotenv.Load()
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
