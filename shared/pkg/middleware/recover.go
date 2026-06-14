package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/wc-fixture/shared/pkg/logger"
)

// Recover captura panics en los handlers y responde 500 en lugar de
// derribar el servidor. Loguea el stack trace completo como error.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log := logger.FromContext(r.Context())
				log.Error("panic recuperado en handler",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				http.Error(w, "error interno del servidor", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
