package main

import (
	"log"
	"os"

	"github.com/gospacex/gpx/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	os.Exit(0)
}
