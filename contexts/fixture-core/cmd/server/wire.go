package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/wc-fixture/fixture-core/internal/application/commands"
	"github.com/wc-fixture/fixture-core/internal/application/queries"
	infrahttp "github.com/wc-fixture/fixture-core/internal/infrastructure/http"
	"github.com/wc-fixture/fixture-core/internal/infrastructure/http/handler"
	infranats "github.com/wc-fixture/fixture-core/internal/infrastructure/nats"
	"github.com/wc-fixture/fixture-core/internal/infrastructure/postgres"
)

// Config contiene toda la configuración del servicio leída desde env vars.
type Config struct {
	DatabaseURL   string
	NATSURL       string
	Port          int
	LogLevel      string
	Env           string
	InternalToken string
}

// App agrupa los componentes de larga vida que deben cerrarse en shutdown.
type App struct {
	Server         *infrahttp.Server
	ResultConsumer *infranats.ResultConsumer
	close          func()
}

// Close libera todos los recursos del App en orden inverso a su inicialización.
func (a *App) Close() {
	if a.close != nil {
		a.close()
	}
}

// wire cablea manualmente todas las dependencias del servicio.
// Orden de inicialización:
//  1. PostgreSQL pool
//  2. NATS connection + JetStream
//  3. Repositorios e infraestructura
//  4. Command handlers
//  5. Query handlers
//  6. HTTP handlers
//  7. Router + Server
//  8. NATS consumers
func wire(ctx context.Context, cfg Config, log *slog.Logger) (*App, error) {
	// ── 1. PostgreSQL ──────────────────────────────────────────────────────────
	pgPool, err := postgres.NewPool(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		return nil, fmt.Errorf("wire: error conectando a PostgreSQL: %w", err)
	}

	// ── 2. NATS JetStream ──────────────────────────────────────────────────────
	natsCfg := infranats.DefaultConfig(cfg.NATSURL)
	nc, js, err := infranats.Connect(ctx, natsCfg)
	if err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("wire: error conectando a NATS: %w", err)
	}

	// ── 3. Repositorios e infraestructura ──────────────────────────────────────
	fixtureRepo    := postgres.NewFixtureRepository(pgPool)
	matchReadModel := postgres.NewMatchReadModel(pgPool)
	standingRM     := postgres.NewStandingReadModel(pgPool)
	eventPublisher := infranats.NewEventPublisher(js)

	// ── 4. Command handlers ────────────────────────────────────────────────────
	initTournamentCmd    := commands.NewInitializeTournamentHandler(fixtureRepo, eventPublisher)
	registerResultCmd    := commands.NewRegisterMatchResultHandler(fixtureRepo, eventPublisher)
	closeGroupStageCmd   := commands.NewCloseGroupStageHandler(fixtureRepo, eventPublisher)
	genKnockoutCmd       := commands.NewGenerateKnockoutBracketHandler(fixtureRepo, eventPublisher)
	advanceKnockoutCmd   := commands.NewAdvanceKnockoutMatchHandler(fixtureRepo, eventPublisher)
	updateScheduleCmd    := commands.NewUpdateMatchScheduleHandler(fixtureRepo, eventPublisher)

	// Suprimir "unused variable" en caso de que no todos los commands
	// tengan endpoint HTTP directo (algunos son solo para uso interno/admin)
	_ = initTournamentCmd
	_ = closeGroupStageCmd
	_ = genKnockoutCmd
	_ = advanceKnockoutCmd

	// ── 5. Query handlers ──────────────────────────────────────────────────────
	// Los read models implementan múltiples interfaces de query
	getFixtureH    := queries.NewGetFixtureHandler(standingRM)
	listGroupsH    := queries.NewListGroupsHandler(standingRM)
	getGroupH      := queries.NewGetGroupHandler(standingRM)
	getStandingsH  := queries.NewGetStandingsHandler(standingRM)
	getBestThirdsH := queries.NewGetBestThirdsHandler(standingRM)
	listMatchesH   := queries.NewListMatchesHandler(matchReadModel)
	getMatchH      := queries.NewGetMatchHandler(matchReadModel)
	getKnockoutH   := queries.NewGetKnockoutHandler(standingRM)
	getKnockoutRoundH := queries.NewGetKnockoutRoundHandler(standingRM)

	// ── 6. HTTP handlers ───────────────────────────────────────────────────────
	fixtureHandler  := handler.NewFixtureHandler(getFixtureH)
	groupHandler    := handler.NewGroupHandler(listGroupsH, getGroupH, getStandingsH, getBestThirdsH)
	matchHandler    := handler.NewMatchHandler(listMatchesH, getMatchH, registerResultCmd, updateScheduleCmd)
	knockoutHandler := handler.NewKnockoutHandler(getKnockoutH, getKnockoutRoundH)

	// ── 7. Router + Server ─────────────────────────────────────────────────────
	router := infrahttp.NewRouter(infrahttp.RouterDeps{
		Logger:          log,
		ServiceName:     "fixture-core",
		FixtureHandler:  fixtureHandler,
		GroupHandler:    groupHandler,
		MatchHandler:    matchHandler,
		KnockoutHandler: knockoutHandler,
	})

	server := infrahttp.NewServer(infrahttp.DefaultServerConfig(cfg.Port), router)

	// ── 8. NATS consumers ──────────────────────────────────────────────────────
	resultConsumer, err := infranats.NewResultConsumer(ctx, js, registerResultCmd, natsCfg.StreamName)
	if err != nil {
		nc.Close()
		pgPool.Close()
		return nil, fmt.Errorf("wire: error creando consumer NATS: %w", err)
	}

	// ── Función de cierre en orden inverso ─────────────────────────────────────
	closeAll := func() {
		log.Info("cerrando recursos de fixture-core")
		nc.Close()
		pgPool.Close()
		log.Info("recursos cerrados")
	}

	return &App{
		Server:         server,
		ResultConsumer: resultConsumer,
		close:          closeAll,
	}, nil
}

// ── Helpers de env vars ────────────────────────────────────────────────────────

// envStr retorna el valor de la env var o el default si está vacía.
func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// envInt retorna el valor entero de la env var o el default si está ausente o inválida.
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
