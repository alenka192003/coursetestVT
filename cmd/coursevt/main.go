package main

import (
	"log"

	"coursevt/internal/config"
	"coursevt/internal/course"
	"coursevt/internal/httpserver"
)

func main() {
	cfg := config.Load()
	store := course.NewMemoryStore()
	server := httpserver.New(cfg, store)

	log.Printf("coursevt service started on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
