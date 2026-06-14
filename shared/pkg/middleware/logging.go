package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/wc-fixture/shared/pkg/logger"
)

// responseWriter wrappea http.ResponseWriter para capturar el status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Status() int {
	if !rw.wroteHeader {
		return http.StatusOK
	}
	return rw.status
}

// Logging registra cada request HTTP con método, path, status y duración.
// Usa el logger inyectado en el contexto (requiere que RequestID vaya antes).
// Los requests a /health se loguean en DEBUG para no saturar los logs.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		log := logger.FromContext(r.Context())
		level := slog.LevelInfo
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
			level = slog.LevelDebug
		}

		log.Log(r.Context(), level, "request completado",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.Status()),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
