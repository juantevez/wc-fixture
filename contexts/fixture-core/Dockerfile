# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Dependencias del sistema para cgo (pgx lo requiere en algunos casos)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Copiar go.mod del módulo shared primero para aprovechar cache de capas
COPY shared/go.mod shared/go.sum ./shared/
COPY contexts/fixture-core/go.mod contexts/fixture-core/go.sum ./contexts/fixture-core/

# Descargar dependencias — cacheado si go.mod no cambió
WORKDIR /src/contexts/fixture-core
RUN go mod download

# Copiar código fuente
WORKDIR /src
COPY shared/ ./shared/
COPY contexts/fixture-core/ ./contexts/fixture-core/

# Build — binario estático para la imagen final mínima
WORKDIR /src/contexts/fixture-core
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /bin/fixture-core ./cmd/server/...

# ── Stage 2: imagen final ─────────────────────────────────────────────────────
FROM scratch

# Certificados CA para conexiones TLS (PostgreSQL, NATS TLS)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Zonas horarias para time.LoadLocation (horarios de partidos)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /bin/fixture-core /fixture-core

EXPOSE 8080

ENTRYPOINT ["/fixture-core"]
