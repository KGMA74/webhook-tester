# webhook-tester

A lightweight, self-hosted webhook inspection tool built with Go's standard library (`net/http`).

Create an instant endpoint, send any HTTP request to it, and inspect headers, query params and body live in a dark UI — no external services, no accounts, no data leaving your machine.

**Live demo → [wt.ryukfearless.digital](https://wt.ryukfearless.digital)**

---

## Features

- **Isolated endpoints** — each user creates their own unique URL (`/hook/<id>`)
- Captures every incoming request — method, headers, query params, body
- Pretty-prints JSON bodies automatically
- Stores the **50 most recent** requests per endpoint (thread-safe, in-memory)
- Live UI via Server-Sent Events — updates instantly, no polling
- Recent endpoints saved in `localStorage` and listed on the home page
- Single static binary — all templates are embedded at compile time
- Optional Basic Auth via `WEBHOOK_TOKEN`

---

## Quick start

```bash
# One-liner with Docker
docker run -d --name webhook-tester -p 8080:8080 ghcr.io/kgma74/webhook-tester:latest
# → http://localhost:8080
```

Or with Docker Compose:

```bash
docker compose up
# → http://localhost:8080
```

---

## How it works

1. Open `http://localhost:8080` → click **Create new endpoint**
2. You get a unique dashboard at `http://localhost:8080/<id>`
3. Configure your provider (Stripe, GitHub, FlowPay…) to send webhooks to:
   ```
   http://your-host/hook/<id>
   ```
4. Requests appear live in the dashboard — expand headers, query params, copy body or curl command

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/` | — | Home — create or reopen an endpoint |
| `GET` | `/new` | — | Generate a new endpoint and redirect |
| `GET` | `/{id}` | optional | Live dashboard for that endpoint |
| `ANY` | `/hook/{id}` | — | Webhook receiver (POST, PUT, GET…) |
| `GET` | `/api/{id}/events` | optional | SSE stream for the dashboard |
| `GET` | `/api/{id}/requests` | optional | JSON list of captured requests |
| `POST` | `/api/{id}/clear` | optional | Flush the endpoint history |
| `GET` | `/health` | — | Health check → `{"status":"ok"}` |

---

## Running locally

**Prerequisites:** Go 1.22+

```bash
git clone https://github.com/KGMA74/webhook-tester
cd webhook-tester
make run
# → http://localhost:8080

# With live reload (requires air)
make watch

# Run tests
make test
```

The port defaults to `8080`. Override with the `PORT` env var or by editing `.env`.

---

## Docker

### Run from GHCR (recommended)

```bash
docker run -d \
  --name webhook-tester \
  -p 8080:8080 \
  ghcr.io/kgma74/webhook-tester:latest
```

### Docker Compose

```bash
docker compose up
# → http://localhost:8080
```

### Build locally

```bash
docker build -t webhook-tester:latest .
docker run -d -p 8080:8080 webhook-tester:latest
```

---

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `PORT` | `8080` | HTTP port to listen on |
| `WEBHOOK_TOKEN` | _(unset)_ | Enable Basic Auth on the dashboard UI (any username, token as password) |

---

## CI/CD — automatic image publishing

Every push to `main` builds and pushes a multi-arch image (`amd64` + `arm64`) to GHCR via GitHub Actions.

| Event | Tags produced |
|-------|---------------|
| Push to `main` | `:main`, `:sha-<short>`, `:latest` |
| Push tag `v1.2.3` | `:1.2.3`, `:1.2`, `:1`, `:latest` |

No secrets to configure — the workflow uses the built-in `GITHUB_TOKEN`.

---

## Testing a capture

```bash
# 1. Create an endpoint (grab the ID from the redirect URL)
curl -si https://wt.ryukfearless.digital/new | grep location
# location: /a3f4b2c18e9d4f01

# 2. Send a webhook to that endpoint
curl -s -X POST https://wt.ryukfearless.digital/hook/a3f4b2c18e9d4f01 \
  -H "Content-Type: application/json" \
  -d '{"event":"payment.success","amount":5000,"currency":"XOF"}'

# 3. Send with query params
curl -s -X POST "https://wt.ryukfearless.digital/hook/a3f4b2c18e9d4f01?token=abc&source=stripe" \
  -d '{"type":"charge.succeeded"}'

# 4. Clear the endpoint history
curl -s -X POST https://wt.ryukfearless.digital/api/a3f4b2c18e9d4f01/clear
```

---

## Project structure

```
webhook-tester/
├── .github/
│   └── workflows/
│       └── release.yml          # Build & push image to GHCR on push/tag
├── cmd/
│   ├── api/
│   │   └── main.go              # Entry point, graceful shutdown
│   └── web/
│       ├── web.go               # Embeds the template directory into the binary
│       └── template/
│           ├── home.html        # Landing page
│           └── index.html       # Per-endpoint live dashboard
├── internal/
│   └── server/
│       ├── endpoint.go          # Endpoint struct + EndpointRegistry
│       ├── server.go            # http.Server construction
│       ├── webhooks.go          # Handlers + WebhookStore
│       ├── sse.go               # SSEBroker (fan-out to connected clients)
│       ├── routes.go            # Route registration
│       └── routes_test.go       # Unit tests
├── Dockerfile                   # Multi-stage build → scratch image
├── docker-compose.yml
└── .env                         # PORT=8080 (not committed in production)
```

---

## Makefile reference

| Command | Description |
|---------|-------------|
| `make build` | Compile to `main.exe` |
| `make run` | Run directly with `go run` |
| `make test` | Run the test suite |
| `make watch` | Live reload via `air` |
| `make clean` | Remove the build artifact |
