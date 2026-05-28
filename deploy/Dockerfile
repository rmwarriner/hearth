# syntax=docker/dockerfile:1

# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependency downloads separately from the build.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /hearthd ./cmd/hearthd

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /hearthd /usr/local/bin/hearthd

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/hearthd"]
