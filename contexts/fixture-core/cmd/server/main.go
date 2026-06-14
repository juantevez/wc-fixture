// main.go es el punto de entrada del servicio fixture-core.
// Responsabilidades:
//  1. Leer configuración desde variables de entorno
//  2. Inicializar infraestructura (PostgreSQL, NATS)
//  3. Cablear dependencias via wire.go
//  4. Iniciar el servidor HTTP y el consumer NATS
//  5. Gestionar el shutdown graceful ante SIGINT/SIGTERM
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
	// Logger de arranque — antes de que esté disponible el logger configurado
	bootLog := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
		With("service", "fixture-core")

	cfg, err := loadConfig()
	if err != nil {
		bootLog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:     cfg.LogLevel,
		Service:   "fixture-core",
		AddSource: cfg.Env == "development",
	})

	// Contexto raíz — se cancela ante SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ctx = logger.WithLogger(ctx, log)

	log.Info("iniciando fixture-core",
		"env", cfg.Env,
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
	)

	// Cablear todas las dependencias
	app, err := wire(ctx, cfg, log)
	if err != nil {
		log.Error("error inicializando dependencias", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	log.Info("dependencias inicializadas correctamente")

	// Iniciar consumer NATS en background
	if err := app.ResultConsumer.Start(ctx); err != nil {
		log.Error("error iniciando consumer NATS", "error", err)
		os.Exit(1)
	}
	log.Info("consumer NATS iniciado")

	// Iniciar servidor HTTP — bloquea hasta que ctx sea cancelado
	if err := app.Server.Start(ctx); err != nil {
		log.Error("error en el servidor HTTP", "error", err)
		os.Exit(1)
	}

	log.Info("fixture-core detenido correctamente")
}

// loadConfig lee la configuración desde variables de entorno.
// Falla rápido si alguna variable requerida está ausente.
func loadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		NATSURL:     os.Getenv("NATS_URL"),
		Port:        envInt("SERVER_PORT", 8080),
		LogLevel:    envStr("LOG_LEVEL", "info"),
		Env:         envStr("ENV", "production"),
		InternalToken: os.Getenv("INTERNAL_TOKEN"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL es requerida")
	}
	if cfg.NATSURL == "" {
		return cfg, fmt.Errorf("NATS_URL es requerida")
	}

	return cfg, nil
}
