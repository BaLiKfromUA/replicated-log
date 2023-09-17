package main

import (
	"log"
	"os"
)

const (
	modePrimary   = "PRIMARY"
	modeSecondary = "SECONDARY"
)

func main() {
	mode, ok := os.LookupEnv("APP_MODE")
	if !ok {
		mode = modePrimary
	}

	switch mode {
	case modePrimary:

	case modeSecondary:
		log.Fatal("Not implemented yet")
	default:
		log.Fatalf("Unexpected mode flag: %s", mode)
	}
}
