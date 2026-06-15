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
		With("service", "notification")

	cfg, err := loadConfig()
	if err != nil {
		bootLog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Service:   "notification",
		AddSource: cfg.Env == "development",
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ctx = logger.WithLogger(ctx, log)
	log.Info("iniciando notification", "env", cfg.Env, "port", cfg.Port)

	app, err := wire(ctx, cfg, log)
	if err != nil {
		log.Error("error inicializando dependencias", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	// Iniciar consumer NATS en background
	if err := app.Consumer.Start(ctx); err != nil {
		log.Error("error iniciando consumer NATS", "error", err)
		os.Exit(1)
	}
	log.Info("consumer NATS iniciado — escuchando fixture.>")

	// Servidor HTTP para health check
	if err := app.Server.Start(ctx); err != nil {
		log.Error("error en el servidor HTTP", "error", err)
		os.Exit(1)
	}

	log.Info("notification detenido correctamente")
}

func loadConfig() (Config, error) {
	cfg := Config{
		NATSURL:  os.Getenv("NATS_URL"),
		Port:     envInt("SERVER_PORT", 8084),
		LogLevel: envStr("LOG_LEVEL", "info"),
		Env:      envStr("ENV", "production"),
	}
	if cfg.NATSURL == "" {
		return cfg, fmt.Errorf("NATS_URL es requerida")
	}
	return cfg, nil
}
