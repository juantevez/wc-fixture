package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/wc-fixture/team-registry/internal/application/queries"
	infrahttp "github.com/wc-fixture/team-registry/internal/infrastructure/http"
	"github.com/wc-fixture/team-registry/internal/infrastructure/http/handler"
	"github.com/wc-fixture/team-registry/internal/infrastructure/postgres"
)

// Config contiene la configuración del servicio.
type Config struct {
	DatabaseURL string
	Port        int
	LogLevel    string
	Env         string
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

func wire(ctx context.Context, cfg Config, log *slog.Logger) (*App, error) {
	// ── 1. PostgreSQL ──────────────────────────────────────────────────────────
	pgPool, err := postgres.NewPool(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		return nil, fmt.Errorf("wire: error conectando a PostgreSQL: %w", err)
	}

	// ── 2. Repositorio ─────────────────────────────────────────────────────────
	teamRepo := postgres.NewTeamRepository(pgPool)

	// ── 3. Query handlers ──────────────────────────────────────────────────────
	getTeamH            := queries.NewGetTeamHandler(teamRepo)
	listTeamsH          := queries.NewListTeamsHandler(teamRepo)
	getConfederationH   := queries.NewGetConfederationHandler(teamRepo)
	listConfederationsH := queries.NewListConfederationsHandler(teamRepo)

	// ── 4. HTTP handler ────────────────────────────────────────────────────────
	teamHandler := handler.NewTeamHandler(
		getTeamH, listTeamsH, getConfederationH, listConfederationsH,
	)

	// ── 5. Router + Server ─────────────────────────────────────────────────────
	router := infrahttp.NewRouter(infrahttp.RouterDeps{
		Logger:      log,
		ServiceName: "team-registry",
		TeamHandler: teamHandler,
	})

	server := infrahttp.NewServer(infrahttp.DefaultServerConfig(cfg.Port), router)

	return &App{
		Server: server,
		close: func() {
			log.Info("cerrando recursos de team-registry")
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
