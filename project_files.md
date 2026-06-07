# Ключевые файлы проекта

В данном файле приведено содержимое основных файлов проекта, важных для реализации приложения, контейнеризации, CI/CD и развертывания в Kubernetes.

Актуальная логика CI/CD: при публикации git-тега `v*` запускаются тесты, сборка Docker-образа и автоматический деплой в локальный Docker Desktop Kubernetes через self-hosted GitHub runner. Для внешнего Kubernetes-кластера дополнительно предусмотрен ручной deploy через `workflow_dispatch`.

## go.mod

```go
module coursevt

go 1.22
```

## cmd/coursevt/main.go

```go
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
```

## internal/config/config.go

```go
package config

import "os"

type Config struct {
	Port    string
	Version string
}

func Load() Config {
	return Config{
		Port:    getenv("PORT", "8080"),
		Version: getenv("APP_VERSION", "dev"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
```

## internal/course/store.go

```go
package course

import "errors"

var ErrNotFound = errors.New("course not found")

type Course struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Hours       int    `json:"hours"`
	Level       string `json:"level"`
}

type Store interface {
	List() []Course
	Get(id string) (Course, error)
}

type MemoryStore struct {
	courses []Course
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		courses: []Course{
			{
				ID:          "go-basics",
				Title:       "Go Basics",
				Description: "Syntax, modules, packages and standard library basics.",
				Hours:       12,
				Level:       "beginner",
			},
			{
				ID:          "docker-kubernetes",
				Title:       "Docker and Kubernetes",
				Description: "Container image build, Kubernetes deployment and ingress setup.",
				Hours:       18,
				Level:       "intermediate",
			},
			{
				ID:          "cicd-github-actions",
				Title:       "CI/CD with GitHub Actions",
				Description: "Tag-based pipelines, image publishing and Kubernetes delivery.",
				Hours:       16,
				Level:       "intermediate",
			},
		},
	}
}

func (s *MemoryStore) List() []Course {
	result := make([]Course, len(s.courses))
	copy(result, s.courses)
	return result
}

func (s *MemoryStore) Get(id string) (Course, error) {
	for _, item := range s.courses {
		if item.ID == id {
			return item, nil
		}
	}

	return Course{}, ErrNotFound
}
```

## internal/httpserver/server.go

```go
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
```

## internal/httpserver/handler.go

```go
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
```

## internal/httpserver/handler_test.go

```go
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
```

## Dockerfile

```dockerfile
FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/coursevt ./cmd/coursevt

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /app/coursevt /app/coursevt

USER app
EXPOSE 8080

ENTRYPOINT ["/app/coursevt"]
```

## .github/workflows/cicd.yml

```yaml
name: CI/CD

on:
  push:
    tags:
      - "v*"
  workflow_dispatch:
    inputs:
      image_tag:
        description: "Image tag to deploy, for example v1.0.0"
        required: true
        type: string

env:
  REGISTRY: ghcr.io
  NAMESPACE: coursevt
  DEPLOYMENT: coursevt
  CONTAINER: coursevt

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Run tests
        run: go test ./...

  build:
    name: Build and push image
    runs-on: ubuntu-latest
    needs: test
    if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')
    permissions:
      contents: read
      packages: write
    outputs:
      image: ${{ steps.meta.outputs.image }}
      version: ${{ steps.meta.outputs.version }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Prepare image metadata
        id: meta
        run: |
          image="${REGISTRY}/${GITHUB_REPOSITORY,,}"
          version="${GITHUB_REF_NAME}"
          echo "image=${image}" >> "$GITHUB_OUTPUT"
          echo "version=${version}" >> "$GITHUB_OUTPUT"

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: |
            ${{ steps.meta.outputs.image }}:${{ steps.meta.outputs.version }}
            ${{ steps.meta.outputs.image }}:latest

  deploy-local:
    name: Deploy to local Kubernetes
    runs-on: self-hosted
    needs: build
    if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Check local Kubernetes access
        run: |
          docker ps
          kubectl config use-context docker-desktop
          kubectl get nodes

      - name: Build local Docker image
        run: docker build -t coursevt:local .

      - name: Load image into Kubernetes node
        run: docker save coursevt:local | docker exec -i desktop-control-plane ctr -n k8s.io images import -

      - name: Apply local manifests
        run: kubectl apply -k k8s/local

      - name: Restart application rollout
        run: |
          kubectl -n "$NAMESPACE" set env deployment/"$DEPLOYMENT" APP_VERSION="$GITHUB_REF_NAME"
          kubectl -n "$NAMESPACE" rollout restart deployment/"$DEPLOYMENT"
          kubectl -n "$NAMESPACE" rollout status deployment/"$DEPLOYMENT" --timeout=120s

      - name: Show deployed resources
        run: |
          kubectl -n "$NAMESPACE" get pods
          kubectl -n "$NAMESPACE" get service
          kubectl -n "$NAMESPACE" get ingress

  deploy-external:
    name: Deploy to external Kubernetes
    runs-on: ubuntu-latest
    needs: test
    if: github.event_name == 'workflow_dispatch' && needs.test.result == 'success'
    environment: production
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Prepare deploy metadata
        id: meta
        run: |
          image="${REGISTRY}/${GITHUB_REPOSITORY,,}"
          version="${{ inputs.image_tag }}"
          echo "image=${image}" >> "$GITHUB_OUTPUT"
          echo "version=${version}" >> "$GITHUB_OUTPUT"

      - name: Set up kubectl
        uses: azure/setup-kubectl@v4

      - name: Configure kubeconfig
        env:
          KUBE_CONFIG_DATA: ${{ secrets.KUBE_CONFIG }}
        run: |
          mkdir -p "$HOME/.kube"
          python3 <<'PY'
          import base64
          import json
          import os
          import subprocess
          import sys

          raw = os.environ["KUBE_CONFIG_DATA"].strip()
          kubeconfig_path = os.path.expanduser("~/.kube/config")

          candidates = [("plain secret", raw)]

          try:
              decoded = base64.b64decode(raw, validate=True).decode("utf-8")
              candidates.append(("base64 secret", decoded.strip()))
          except Exception:
              pass

          for name, value in list(candidates):
              try:
                  parsed = json.loads(value)
                  if isinstance(parsed, str):
                      candidates.append((f"{name} as JSON string", parsed.strip()))
                  elif isinstance(parsed, dict):
                      candidates.append((f"{name} as JSON object", json.dumps(parsed)))
              except Exception:
                  pass

          for name, value in candidates:
              with open(kubeconfig_path, "w", encoding="utf-8") as file:
                  file.write(value)
                  file.write("\n")

              result = subprocess.run(
                  ["kubectl", "config", "view", "--kubeconfig", kubeconfig_path],
                  stdout=subprocess.DEVNULL,
                  stderr=subprocess.DEVNULL,
                  check=False,
              )
              if result.returncode == 0:
                  print(f"Kubeconfig written from {name}")
                  sys.exit(0)

          print("KUBE_CONFIG is not a valid kubeconfig", file=sys.stderr)
          sys.exit(1)
          PY
          chmod 600 "$HOME/.kube/config"

      - name: Apply manifests
        run: kubectl apply -k k8s/base

      - name: Update application image
        run: |
          kubectl -n "$NAMESPACE" set image deployment/"$DEPLOYMENT" "$CONTAINER"="${{ steps.meta.outputs.image }}:${{ steps.meta.outputs.version }}"
          kubectl -n "$NAMESPACE" set env deployment/"$DEPLOYMENT" APP_VERSION="${{ steps.meta.outputs.version }}"
          kubectl -n "$NAMESPACE" rollout status deployment/"$DEPLOYMENT" --timeout=120s
```

## Kubernetes: k8s/base/namespace.yaml

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: coursevt
```

## Kubernetes: k8s/base/deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coursevt
  namespace: coursevt
  labels:
    app: coursevt
spec:
  replicas: 2
  selector:
    matchLabels:
      app: coursevt
  template:
    metadata:
      labels:
        app: coursevt
    spec:
      containers:
        - name: coursevt
          image: ghcr.io/owner/coursevt:v0.1.0
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 8080
          env:
            - name: PORT
              value: "8080"
            - name: APP_VERSION
              value: "v0.1.0"
          readinessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 10
            periodSeconds: 20
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 250m
              memory: 128Mi
```

## Kubernetes: k8s/base/service.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  name: coursevt
  namespace: coursevt
  labels:
    app: coursevt
spec:
  selector:
    app: coursevt
  ports:
    - name: http
      port: 80
      targetPort: http
  type: ClusterIP
```

## Kubernetes: k8s/base/ingress.yaml

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: coursevt
  namespace: coursevt
  labels:
    app: coursevt
spec:
  ingressClassName: nginx
  rules:
    - host: coursevt.traefik.me
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: coursevt
                port:
                  number: 80
```

## Kubernetes: k8s/base/kustomization.yaml

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - namespace.yaml
  - deployment.yaml
  - service.yaml
  - ingress.yaml
```

## Kubernetes local: k8s/local/kustomization.yaml

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../base
patches:
  - path: deployment-local.yaml
  - path: ingress-local.yaml
```

## Kubernetes local: k8s/local/deployment-local.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coursevt
  namespace: coursevt
spec:
  template:
    spec:
      containers:
        - name: coursevt
          image: coursevt:local
          imagePullPolicy: Never
          env:
            - name: PORT
              value: "8080"
            - name: APP_VERSION
              value: "local"
```

## Kubernetes local: k8s/local/ingress-local.yaml

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: coursevt
  namespace: coursevt
spec:
  ingressClassName: nginx
  rules:
    - host: coursevt.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: coursevt
                port:
                  number: 80
```
