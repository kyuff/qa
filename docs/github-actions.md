# GitHub Actions Integration

This guide shows how to wire `qa` into a GitHub Actions workflow using Docker Compose to
run the full integration suite in CI.

## How it works in CI

| Step | What runs | `QA_MODE` |
|------|-----------|-----------|
| Unit tests | `go test ./internal/...` | *(unset)* |
| Build image | `docker build` | — |
| Compose up | postgres + stubs container + app container | `stubs-only` (inside stubs container) |
| QA tests | `go test ./tests/...` on the runner | `ci` |

The stubs container runs the compiled test binary in `stubs-only` mode, which starts the
control server and all registered stubs but neither the app nor the tests. Once everything
is healthy, the runner executes the tests in `ci` mode, which proxies all stub interactions
to the stubs container and sends a shutdown signal when the tests are done.

## File layout

```
.
├── cmd/myapp/          # application entry point
├── internal/           # packages with unit-tested business logic
├── tests/
│   └── main_test.go    # TestMain — wires qa.Run
├── Dockerfile          # builds the app image
├── Dockerfile.qa       # builds the test binary for the stubs container
└── docker-compose.qa.yml
```

## tests/main_test.go

Stub addresses are fixed ports so that the stubs container and the app container can agree
on them without runtime coordination. `WithControlAddrEnv` lets CI override the bind address
(`0.0.0.0:9000` in the container) while local development falls back to `localhost:9000`.

```go
package tests

import (
    "os"
    "testing"

    "github.com/kyuff/qa"
    "github.com/kyuff/qa/httpstub"
)

var paymentStub = httpstub.New(
    httpstub.WithAddr("0.0.0.0:19001"), // fixed port, listens on all interfaces
)

func TestMain(m *testing.M) {
    os.Exit(qa.Run(m,
        qa.WithStub("payments", paymentStub),

        // QA_CONTROL_ADDR overrides the local default.
        // stubs container:  QA_CONTROL_ADDR=0.0.0.0:9000  (bind on all interfaces)
        // CI test step:     QA_CONTROL_ADDR=localhost:9000 (connect via published port)
        qa.WithControlAddrEnv("QA_CONTROL_ADDR"),
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

Standard multi-stage build for the application:

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

## Dockerfile.qa

Compiles the test binary and packages it as a container for the stubs container:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go test -c -o /qa-tests ./tests/

FROM alpine:3.19
RUN apk add --no-cache curl
COPY --from=builder /qa-tests /qa-tests
ENTRYPOINT ["/qa-tests"]
```

## docker-compose.qa.yml

Starts postgres, the stubs container, and the application together.
The app connects to the stubs via the internal Docker network (`stubs:19001`).
The CI runner connects to the control server via the published port (`localhost:9000`).

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

  stubs:
    build:
      context: .
      dockerfile: Dockerfile.qa
    environment:
      QA_MODE: stubs-only
      QA_CONTROL_ADDR: "0.0.0.0:9000"
    ports:
      - "9000:9000"     # control server — reached by the CI test step on the runner
      - "19001:19001"   # payment stub — published for local debugging; app uses the internal address
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/_qa/health"]
      interval: 2s
      timeout: 5s
      retries: 15

  app:
    image: myapp:${IMAGE_TAG}
    environment:
      DATABASE_URL: postgres://myapp:secret@postgres:5432/myapp?sslmode=disable
      PAYMENT_STUB_URL: http://stubs:19001
    depends_on:
      postgres:
        condition: service_healthy
      stubs:
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

      - name: Start infrastructure, stubs and app
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

With no environment variables set, `qa.Run` defaults to local mode: stubs start, then the app
is started via `WithAppCmd`, then the tests run, and everything shuts down in order.

```sh
go test ./tests/ -count 1 -v
```
