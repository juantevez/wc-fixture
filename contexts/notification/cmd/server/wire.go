package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/wc-fixture/notification/internal/application/handlers"
	infrahttp "github.com/wc-fixture/notification/internal/infrastructure/http"
	"github.com/wc-fixture/notification/internal/infrastructure/http/handler"
	infranats "github.com/wc-fixture/notification/internal/infrastructure/nats"
	"github.com/wc-fixture/notification/internal/infrastructure/webhook"
)

// Config contiene la configuración del servicio notification.
// Nota: notification NO necesita PostgreSQL — no tiene estado propio.
// Los suscriptores webhook se gestionarían en un futuro con su propia DB,
// pero en esta fase se usa un repositorio en memoria o configuración estática.
type Config struct {
	NATSURL  string
	Port     int
	LogLevel string
	Env      string
}

// App agrupa los componentes de larga vida.
type App struct {
	Server   *infrahttp.Server
	Consumer *infranats.FixtureConsumer
	close    func()
}

func (a *App) Close() {
	if a.close != nil {
		a.close()
	}
}

// wire cablea todas las dependencias de notification.
func wire(ctx context.Context, cfg Config, log *slog.Logger) (*App, error) {
	// ── 1. NATS JetStream ──────────────────────────────────────────────────────
	natsCfg := infranats.DefaultConfig(cfg.NATSURL)
	nc, js, err := infranats.Connect(ctx, natsCfg)
	if err != nil {
		return nil, fmt.Errorf("wire: error conectando a NATS: %w", err)
	}

	// ── 2. Repositorio de webhooks (en memoria para esta fase) ─────────────────
	// En producción esto se reemplaza por un PostgreSQL repo.
	webhookRepo := newInMemoryWebhookRepo()

	// ── 3. Notifier (webhook dispatcher) ──────────────────────────────────────
	webhookDispatcher := webhook.NewDispatcher(webhookRepo)

	// ── 4. Event handlers ──────────────────────────────────────────────────────
	matchResultH      := handlers.NewMatchResultHandler(webhookDispatcher)
	bracketGeneratedH := handlers.NewBracketGeneratedHandler(webhookDispatcher)
	tournamentFinishedH := handlers.NewTournamentFinishedHandler(webhookDispatcher)

	dispatcher := handlers.NewEventDispatcher(
		matchResultH,
		bracketGeneratedH,
		tournamentFinishedH,
	)

	// ── 5. NATS consumer ───────────────────────────────────────────────────────
	consumer, err := infranats.NewFixtureConsumer(ctx, js, dispatcher, natsCfg.StreamName)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("wire: error creando consumer NATS: %w", err)
	}

	// ── 6. HTTP server (solo health check) ─────────────────────────────────────
	router := infrahttp.NewRouter(infrahttp.RouterDeps{
		Logger:        log,
		ServiceName:   "notification",
		HealthHandler: handler.NewHealthHandler(),
	})

	server := infrahttp.NewServer(infrahttp.DefaultServerConfig(cfg.Port), router)

	return &App{
		Server:   server,
		Consumer: consumer,
		close: func() {
			log.Info("cerrando recursos de notification")
			nc.Close()
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
