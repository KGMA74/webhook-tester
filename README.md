# webhook-tester

A lightweight, self-hosted webhook inspection tool built with Go's standard library (`net/http`).

Send any HTTP POST to `/webhook` and instantly inspect the captured headers and body in a live-updating dark UI — no external services, no accounts, no data leaving your machine.

---

## Features

- Captures every `POST /webhook` — headers, method, timestamp, and body
- Pretty-prints JSON bodies automatically
- Stores the **50 most recent** requests in memory (thread-safe, zero persistence)
- Live UI that auto-refreshes every 2 seconds without a full page reload
- `POST /clear` endpoint to flush the history
- Single static binary — all templates are embedded at compile time

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Dashboard UI |
| `POST` | `/webhook` | Capture a webhook (any body) |
| `POST` | `/clear` | Flush the in-memory history |
| `GET` | `/api/requests` | JSON feed used by the live UI |

---

## Running locally

**Prerequisites:** Go 1.22+

```bash
# Clone and run
git clone <repo-url>
cd webhook-tester
make run
# → http://localhost:8080

# With live reload (requires air)
make watch

# Run tests
make test
```

The port defaults to `8080`. Override it with the `PORT` env var or by editing `.env`.

---

## Docker

### Build and run with Docker Compose

```bash
docker compose up --build
# → http://localhost:8111
```

The compose file maps host port **8111** → container port **8080**. Edit `docker-compose.yml` to change it.

### Build the image manually

```bash
docker build -t webhook-tester:latest .
```

### Run the container directly

```bash
docker run -d \
  --name webhook-tester \
  -p 8111:8080 \
  -e PORT=8080 \
  webhook-tester:latest
```

### Publish to a registry

```bash
docker tag webhook-tester:latest ghcr.io/<your-org>/webhook-tester:latest
docker push ghcr.io/<your-org>/webhook-tester:latest
```

Then replace the `build` block in `docker-compose.yml` with:

```yaml
image: ghcr.io/<your-org>/webhook-tester:latest
```

---

## Testing a capture

```bash
# Plain JSON payload
curl -s -X POST http://localhost:8111/webhook \
  -H "Content-Type: application/json" \
  -d '{"event":"payment.success","amount":5000,"currency":"XOF"}'

# Plain text payload
curl -s -X POST http://localhost:8111/webhook \
  -H "Content-Type: text/plain" \
  -d 'ping'

# Clear the history
curl -s -X POST http://localhost:8111/clear
```

---

## Project structure

```
webhook-tester/
├── cmd/
│   ├── api/
│   │   └── main.go              # Entry point, graceful shutdown
│   └── web/
│       ├── web.go               # Embeds the template directory into the binary
│       └── template/
│           └── index.html       # Dashboard UI (Tailwind CSS CDN)
├── internal/
│   └── server/
│       ├── server.go            # http.Server construction, template parsing
│       ├── webhooks.go          # WebhookStore, handlers (index, webhook, clear, api)
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
