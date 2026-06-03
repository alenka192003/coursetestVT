package httpserver

import (
	"net/http"
	"time"

	"coursevt/internal/config"
	"coursevt/internal/course"
)

func New(cfg config.Config, courses course.Store) *http.Server {
	h := NewHandler(cfg, courses)

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
}
