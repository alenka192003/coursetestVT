# Курсовой проект: Go CI/CD в Kubernetes

Проект демонстрирует CI/CD-конвейер для Go HTTP API: сборка запускается при появлении git-тега, Docker-образ публикуется в GitHub Container Registry, после чего приложение разворачивается в Kubernetes и становится доступно через Ingress по доменному имени.

## Что реализовано

- HTTP API на Go, обрабатывающее GET-запросы.
- Архитектура разделена на `cmd/` и `internal/` пакеты.
- Endpoint `/` возвращает информацию о сервисе, версии и доступных маршрутах.
- Endpoint `/healthz` используется Kubernetes readiness/liveness probes.
- Endpoint `/api/v1/courses` возвращает список учебных курсов.
- Endpoint `/api/v1/courses/{id}` возвращает курс по идентификатору.
- Dockerfile для сборки production-образа.
- Kubernetes-манифесты: Namespace, Deployment, Service, Ingress.
- GitHub Actions workflow `.github/workflows/cicd.yml`.
- Автоматический запуск сборки при создании git-тега вида `v*`.
- Автоматический деплой после успешной сборки или ручной деплой через `workflow_dispatch`.

## Структура

```text
.
├── .github/workflows/cicd.yml
├── cmd/coursevt/main.go
├── internal/
│   ├── config/config.go
│   ├── course/store.go
│   └── httpserver/
│       ├── handler.go
│       ├── handler_test.go
│       └── server.go
├── k8s/
│   ├── deployment.yaml
│   ├── ingress.yaml
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   └── service.yaml
├── Dockerfile
├── go.mod
└── README.md
```

## Архитектура приложения

- `cmd/coursevt/main.go` содержит только точку входа, загрузку конфигурации и запуск сервера.
- `internal/config` отвечает за чтение переменных окружения.
- `internal/course` содержит модель курса и in-memory хранилище.
- `internal/httpserver` содержит маршруты, обработчики, JSON-ответы и HTTP-сервер.

## Локальный запуск

```bash
go test ./...
go run ./cmd/coursevt
```

Проверка:

```bash
curl http://localhost:8080/
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/courses
curl http://localhost:8080/api/v1/courses/go-basics
```

## API

### `GET /`

Возвращает описание сервиса и список доступных endpoint'ов.

### `GET /healthz`

Возвращает статус работоспособности сервиса.

```json
{
  "status": "ok"
}
```

### `GET /api/v1/courses`

Возвращает список курсов.

```json
{
  "count": 3,
  "items": [
    {
      "id": "go-basics",
      "title": "Go Basics",
      "description": "Syntax, modules, packages and standard library basics.",
      "hours": 12,
      "level": "beginner"
    }
  ]
}
```

### `GET /api/v1/courses/{id}`

Возвращает один курс по идентификатору.

## Настройка Kubernetes

В кластере должен быть установлен Ingress Controller. В манифесте используется `ingressClassName: nginx`. Если в вашем кластере используется другой ingress class, измените поле в `k8s/ingress.yaml`.

Текущий домен приложения:

```text
coursevt.traefik.me
```

Если используется внешний кластер, удобнее заменить host на домен nip.io, привязанный к внешнему IP ingress controller:

```text
coursevt.<EXTERNAL-IP>.nip.io
```

Например:

```text
coursevt.203.0.113.10.nip.io
```

## Настройка GitHub Actions

Workflow использует GitHub Container Registry: `ghcr.io/<owner>/<repo>`.

Нужно добавить secret в GitHub репозитории:

```text
KUBE_CONFIG
```

Значение секрета может быть обычным содержимым kubeconfig YAML или base64-представлением kubeconfig:

```bash
base64 -w 0 ~/.kube/config
```

Если GHCR package останется приватным, в Kubernetes нужно добавить `imagePullSecret` или сделать package публичным. Для учебного проекта проще сделать опубликованный образ публичным в настройках GHCR package.

## Запуск CI/CD по тегу

После создания удалённого GitHub-репозитория выполните:

```bash
git remote add origin git@github.com:<owner>/<repo>.git
git push -u origin main
git tag v1.0.0
git push origin v1.0.0
```

После push тега workflow выполнит:

1. Запуск тестов Go.
2. Сборку Docker-образа.
3. Публикацию образа в GHCR.
4. Применение Kubernetes-манифестов.
5. Обновление образа в Deployment.
6. Ожидание успешного rollout.

## Ручной деплой

В GitHub Actions можно запустить workflow вручную через `Run workflow` и указать тег образа, например:

```text
v1.0.0
```

Если для environment `production` включить required reviewers, деплой после сборки станет ручным подтверждаемым шагом.

## Проверка после деплоя

```bash
kubectl -n coursevt get pods
kubectl -n coursevt get ingress
curl http://coursevt.traefik.me/
curl http://coursevt.traefik.me/api/v1/courses
```

Ожидаемый ответ:

```json
{
  "service": "coursevt",
  "message": "Go CI/CD course project API",
  "version": "v1.0.0",
  "endpoints": [
    "GET /",
    "GET /healthz",
    "GET /api/v1/courses",
    "GET /api/v1/courses/{id}"
  ],
  "timestamp": "2026-06-03T12:00:00Z"
}
```

## Соответствие требованиям

- Приложение обрабатывает HTTP GET-запросы: реализованы `/`, `/healthz`, `/api/v1/courses`, `/api/v1/courses/{id}`.
- Код расположен в git-репозитории: проект предназначен для публикации в GitHub.
- Сборка запускается при появлении git-тега: trigger `push.tags: v*`.
- Доставка в Kubernetes выполняется автоматически после сборки или вручную через `workflow_dispatch`.
- Доступ по доменному имени выполняется через Kubernetes Ingress: `coursevt.traefik.me` или nip.io host.
