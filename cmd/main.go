package main

import (
	"log"
	"net/http"
	"os"
	"replicated-log/internal/secondary"
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

	var srv *http.Server

	switch mode {
	case modePrimary:
		log.Fatal("Not implemented yet")
	case modeSecondary:
		srv = secondary.NewSecondaryServer()
	default:
		log.Fatalf("Unexpected mode flag: %s", mode)
	}

	log.Printf("Start serving HTTP at %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
