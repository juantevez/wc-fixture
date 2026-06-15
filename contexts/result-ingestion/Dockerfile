# =============================================================================
# Dockerfile — result-ingestion
# Build context: raíz del monorepo (wc-fixture/)
# =============================================================================

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum                                     ./shared/
COPY contexts/fixture-core/go.mod contexts/fixture-core/go.sum       ./contexts/fixture-core/
COPY contexts/venue-geo/go.mod contexts/venue-geo/go.sum             ./contexts/venue-geo/
COPY contexts/team-registry/go.mod contexts/team-registry/go.sum     ./contexts/team-registry/
COPY contexts/result-ingestion/go.mod contexts/result-ingestion/go.sum ./contexts/result-ingestion/
COPY contexts/notification/go.mod contexts/notification/go.sum       ./contexts/notification/

RUN go work sync

COPY shared/                    ./shared/
COPY contexts/fixture-core/     ./contexts/fixture-core/
COPY contexts/venue-geo/        ./contexts/venue-geo/
COPY contexts/team-registry/    ./contexts/team-registry/
COPY contexts/result-ingestion/ ./contexts/result-ingestion/
COPY contexts/notification/     ./contexts/notification/

WORKDIR /src/contexts/result-ingestion
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /bin/result-ingestion ./cmd/server/...

# ── Stage 2: imagen final ─────────────────────────────────────────────────────
# alpine en lugar de scratch — provee sh, wget y tzdata para el healthcheck
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /bin/result-ingestion /result-ingestion

EXPOSE 8083

ENTRYPOINT ["/result-ingestion"]
