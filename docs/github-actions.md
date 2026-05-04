# GitHub Actions Integration

This guide shows how to wire `qa` into a GitHub Actions workflow. The stubs run directly
on the CI runner as a background process, so no extra Dockerfile is needed.

## How it works in CI

| Step | What runs | `QA_MODE` |
|------|-----------|-----------|
| Unit tests | `go test ./internal/...` | *(unset)* |
| Build image | `docker build` | — |
| Start stubs | compiled test binary, background | `stubs-only` |
| Compose up | postgres + app container | — |
| QA tests | `go test ./tests/...` on the runner | `ci` |

The stubs process binds on all interfaces so the app container can reach it via
`host.docker.internal`. The control server listens on `localhost:9000`, which is shared by
both the stubs process and the CI test step since they both run on the same runner.

## File layout

```
.
├── cmd/myapp/          # application entry point
├── internal/           # packages with unit-tested business logic
├── tests/
│   └── main_test.go    # TestMain — wires qa.Run
├── Dockerfile          # builds the app image
└── docker-compose.qa.yml
```

## tests/main_test.go

```go
package tests

import (
    "os"
    "testing"

    "github.com/kyuff/qa"
    "github.com/kyuff/qa/httpstub"
)

var paymentStub = httpstub.New(
    httpstub.WithAddr("0.0.0.0:19001"), // fixed port, all interfaces so Docker can reach it
)

func TestMain(m *testing.M) {
    os.Exit(qa.Run(m,
        qa.WithStub("payments", paymentStub),
        qa.WithControlAddr("localhost:9000"),

        // WithAppCmd and WithAppHealthCheck are only used in local mode.
        // In CI, docker compose starts the app.
        qa.WithAppCmd("go", "run", "./cmd/myapp",
            "--payment-url=http://localhost:19001",
        ),
        qa.WithAppHealthCheck("http://localhost:8080/health"),
    ))
}
```

## Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /myapp ./cmd/myapp/

FROM alpine:3.19
RUN apk add --no-cache curl
COPY --from=builder /myapp /myapp
EXPOSE 8080
ENTRYPOINT ["/myapp"]
```

## docker-compose.qa.yml

The app reaches the stubs on the runner via `host.docker.internal`. The `host-gateway`
extra host entry is required on Linux (which GitHub Actions uses).

```yaml
name: myapp-qa

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: myapp
      POSTGRES_PASSWORD: secret
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "myapp", "-d", "myapp"]
      interval: 2s
      timeout: 5s
      retries: 10

  app:
    image: myapp:${IMAGE_TAG}
    extra_hosts:
      - "host.docker.internal:host-gateway"
    environment:
      DATABASE_URL: postgres://myapp:secret@postgres:5432/myapp?sslmode=disable
      PAYMENT_STUB_URL: http://host.docker.internal:19001
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 2s
      timeout: 10s
      retries: 15
```

## .github/workflows/qa.yml

```yaml
name: QA

on:
  push:
  pull_request:

jobs:
  unit:
    name: Unit tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true
      - name: Test
        run: go test ./internal/... -count 1 -race

  qa:
    name: QA tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true

      - name: Build app image
        run: docker build -t myapp:${{ github.sha }} .

      - name: Build test binary
        run: go test -c -o qa-tests ./tests/

      - name: Start stubs
        env:
          QA_MODE: stubs-only
          QA_CONTROL_ADDR: localhost:9000
        run: ./qa-tests &

      - name: Wait for stubs
        run: |
          for i in $(seq 1 30); do
            curl -sf http://localhost:9000/_qa/health && break || sleep 1
          done

      - name: Start infrastructure and app
        env:
          IMAGE_TAG: ${{ github.sha }}
        run: docker compose -f docker-compose.qa.yml up -d --wait

      - name: Run QA tests
        env:
          QA_MODE: ci
          QA_CONTROL_ADDR: localhost:9000
        run: go test ./tests/ -count 1 -v -timeout 5m

      - name: Collect logs on failure
        if: failure()
        run: docker compose -f docker-compose.qa.yml logs

      - name: Teardown
        if: always()
        run: docker compose -f docker-compose.qa.yml down -v
```

## Running locally

With no environment variables set, `qa.Run` defaults to local mode: stubs start, then the
app is started via `WithAppCmd`, then the tests run, and everything shuts down in order.

```sh
go test ./tests/ -count 1 -v
```
