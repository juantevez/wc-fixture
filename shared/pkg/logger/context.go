package logger

import (
	"context"
	"log/slog"
)

type contextKey struct{}

// WithLogger almacena el logger en el contexto.
// Los handlers HTTP lo inyectan al inicio de cada request.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext extrae el logger del contexto.
// Si no hay logger en el contexto retorna un logger de fallback con INFO.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// WithFields retorna un nuevo logger (desde el contexto) con los campos dados.
// Conveniente para agregar campos de dominio sin propagar el logger manualmente.
//
//	log := logger.WithFields(ctx, "match_id", matchID, "phase", phase)
//	log.Info("resultado registrado")
func WithFields(ctx context.Context, args ...any) *slog.Logger {
	return FromContext(ctx).With(args...)
}
