package main

import (
	"log"
	"net/http"

	"campaign-lottery-platform/backend/internal/config"
	"campaign-lottery-platform/backend/internal/router"
)

func main() {
	cfg := config.Load()
	handler, err := router.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("campaign lottery api listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handler); err != nil {
		log.Fatal(err)
	}
}
