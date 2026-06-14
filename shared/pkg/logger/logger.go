// Package logger provee un logger estructurado basado en slog con formato JSON.
// Todos los servicios del monorepo usan este paquete para garantizar un formato
// de log uniforme con campos mínimos requeridos: service, level, msg, time.
package logger

import (
	"log/slog"
	"os"
)

// Config contiene las opciones de inicialización del logger.
type Config struct {
	// Level es el nivel mínimo de log: "debug", "info", "warn", "error".
	// Por defecto "info".
	Level string

	// Service es el nombre del bounded context que inicializa el logger.
	// Aparece como campo "service" en todos los registros.
	Service string

	// AddSource agrega el archivo y línea de llamada a cada registro.
	// Útil en desarrollo, costoso en producción.
	AddSource bool
}

// New crea un *slog.Logger con salida JSON a stdout.
// Siempre incluye el campo "service" en todos los registros.
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	})

	return slog.New(handler).With(slog.String("service", cfg.Service))
}

// NewDefault crea un logger con nivel INFO, sin source, útil para tests
// o contextos donde no se necesita configuración explícita.
func NewDefault(service string) *slog.Logger {
	return New(Config{Level: "info", Service: service})
}

// parseLevel convierte el string de nivel a slog.Level.
// Si el string no es reconocido, retorna INFO.
func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
