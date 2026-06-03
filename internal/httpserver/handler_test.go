package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"coursevt/internal/config"
	"coursevt/internal/course"
)

func TestIndex(t *testing.T) {
	rec := performRequest(http.MethodGet, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Go CI/CD course project API") {
		t.Fatalf("expected service message in response, got %q", rec.Body.String())
	}
}

func TestHealth(t *testing.T) {
	rec := performRequest(http.MethodGet, "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("expected health response, got %q", rec.Body.String())
	}
}

func TestListCourses(t *testing.T) {
	rec := performRequest(http.MethodGet, "/api/v1/courses")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "go-basics") {
		t.Fatalf("expected course list response, got %q", rec.Body.String())
	}
}

func TestGetCourse(t *testing.T) {
	rec := performRequest(http.MethodGet, "/api/v1/courses/docker-kubernetes")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Docker and Kubernetes") {
		t.Fatalf("expected course response, got %q", rec.Body.String())
	}
}

func TestGetCourseNotFound(t *testing.T) {
	rec := performRequest(http.MethodGet, "/api/v1/courses/unknown")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRejectsPost(t *testing.T) {
	rec := performRequest(http.MethodPost, "/api/v1/courses")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func performRequest(method, path string) *httptest.ResponseRecorder {
	h := NewHandler(config.Config{Port: "8080", Version: "test"}, course.NewMemoryStore())
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}
