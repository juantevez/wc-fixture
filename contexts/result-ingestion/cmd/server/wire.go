package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/result-ingestion/internal/application/commands"
	infrahttp "github.com/wc-fixture/result-ingestion/internal/infrastructure/http"
	"github.com/wc-fixture/result-ingestion/internal/infrastructure/http/handler"
	"github.com/wc-fixture/result-ingestion/internal/infrastructure/idempotency"
	infranats "github.com/wc-fixture/result-ingestion/internal/infrastructure/nats"
)

// Config contiene la configuración del servicio.
type Config struct {
	DatabaseURL   string
	NATSURL       string
	Port          int
	LogLevel      string
	Env           string
	InternalToken string
}

// App agrupa los componentes de larga vida.
type App struct {
	Server *infrahttp.Server
	close  func()
}

func (a *App) Close() {
	if a.close != nil {
		a.close()
	}
}

// wire cablea todas las dependencias de result-ingestion.
func wire(ctx context.Context, cfg Config, log *slog.Logger) (*App, error) {
	// ── 1. PostgreSQL ──────────────────────────────────────────────────────────
	pgCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("wire: config PostgreSQL inválida: %w", err)
	}
	pgCfg.MaxConns = 10

	pgPool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return nil, fmt.Errorf("wire: error conectando a PostgreSQL: %w", err)
	}
	if err := pgPool.Ping(ctx); err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("wire: ping PostgreSQL fallido: %w", err)
	}

	// ── 2. NATS JetStream ──────────────────────────────────────────────────────
	natsCfg := infranats.DefaultConfig(cfg.NATSURL)
	nc, js, err := infranats.Connect(ctx, natsCfg)
	if err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("wire: error conectando a NATS: %w", err)
	}

	// ── 3. Infraestructura ─────────────────────────────────────────────────────
	publisher      := infranats.NewEventPublisher(js)
	idempotencyStore := idempotency.NewPostgresStore(pgPool)

	// ── 4. Command handler ─────────────────────────────────────────────────────
	ingestResultH := commands.NewIngestResultHandler(publisher, idempotencyStore)

	// ── 5. HTTP handler ────────────────────────────────────────────────────────
	resultHandler := handler.NewResultHandler(ingestResultH)

	// ── 6. Router + Server ─────────────────────────────────────────────────────
	router := infrahttp.NewRouter(infrahttp.RouterDeps{
		Logger:        log,
		ServiceName:   "result-ingestion",
		ResultHandler: resultHandler,
		InternalToken: cfg.InternalToken,
	})

	server := infrahttp.NewServer(infrahttp.DefaultServerConfig(cfg.Port), router)

	return &App{
		Server: server,
		close: func() {
			log.Info("cerrando recursos de result-ingestion")
			nc.Close()
			pgPool.Close()
		},
	}, nil
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
