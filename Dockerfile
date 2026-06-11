# ── Stage 1: build ────────────────────────────────────────────────────────────
# Use the Alpine variant for a minimal builder image.
# Pin the minor version; adjust when upgrading the go.mod toolchain directive.
FROM golang:1.26-alpine AS builder

# git is required by some go modules that fetch via VCS.
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache the module graph before copying sources so Docker reuses this layer
# on every build that doesn't change go.mod / go.sum.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source tree.
COPY . .

# Produce a fully static binary:
#   CGO_ENABLED=0  → no libc dependency
#   -trimpath      → strip module paths from stack traces (smaller + more reproducible)
#   -ldflags="-s -w" → strip symbol table and DWARF debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/webhook-tester ./cmd/api

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
# scratch gives us the smallest possible attack surface.
# All templates are embedded in the binary, so no extra files are needed.
FROM scratch

# Import TLS root certificates (needed for any future outbound HTTPS calls)
# and timezone data (for correct UTC/local time formatting).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /out/webhook-tester /webhook-tester

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/webhook-tester"]
