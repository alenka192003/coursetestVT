package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"coursevt/internal/config"
	"coursevt/internal/course"
)

type Handler struct {
	config  config.Config
	courses course.Store
}

type serviceInfo struct {
	Service   string   `json:"service"`
	Message   string   `json:"message"`
	Version   string   `json:"version"`
	Endpoints []string `json:"endpoints"`
	Timestamp string   `json:"timestamp"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(cfg config.Config, courses course.Store) *Handler {
	return &Handler{config: cfg, courses: courses}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.index)
	mux.HandleFunc("/healthz", h.health)
	mux.HandleFunc("/api/v1/courses", h.listCourses)
	mux.HandleFunc("/api/v1/courses/", h.getCourse)
	return mux
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		notFound(w, "route not found")
		return
	}

	if !requireGET(w, r) {
		return
	}

	writeJSON(w, http.StatusOK, serviceInfo{
		Service: "coursevt",
		Message: "Go CI/CD course project API",
		Version: h.config.Version,
		Endpoints: []string{
			"GET /",
			"GET /healthz",
			"GET /api/v1/courses",
			"GET /api/v1/courses/{id}",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listCourses(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/courses" {
		notFound(w, "route not found")
		return
	}

	if !requireGET(w, r) {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": h.courses.List(),
		"count": len(h.courses.List()),
	})
}

func (h *Handler) getCourse(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/courses/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		notFound(w, "course not found")
		return
	}

	item, err := h.courses.Get(id)
	if err != nil {
		if errors.Is(err, course.ErrNotFound) {
			notFound(w, "course not found")
			return
		}

		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func requireGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}

	w.Header().Set("Allow", http.MethodGet)
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
	return false
}

func notFound(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusNotFound, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
