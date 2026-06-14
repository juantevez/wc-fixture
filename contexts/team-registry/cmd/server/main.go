package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/wc-fixture/shared/pkg/logger"
)

func main() {
	bootLog := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With("service", "team-registry")

	cfg, err := loadConfig()
	if err != nil {
		bootLog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Service:   "team-registry",
		AddSource: cfg.Env == "development",
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ctx = logger.WithLogger(ctx, log)
	log.Info("iniciando team-registry", "env", cfg.Env, "port", cfg.Port)

	app, err := wire(ctx, cfg, log)
	if err != nil {
		log.Error("error inicializando dependencias", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := app.Server.Start(ctx); err != nil {
		log.Error("error en el servidor HTTP", "error", err)
		os.Exit(1)
	}

	log.Info("team-registry detenido correctamente")
}

func loadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        envInt("SERVER_PORT", 8082),
		LogLevel:    envStr("LOG_LEVEL", "info"),
		Env:         envStr("ENV", "production"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL es requerida")
	}
	return cfg, nil
}
