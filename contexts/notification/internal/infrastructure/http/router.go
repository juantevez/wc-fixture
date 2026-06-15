package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wc-fixture/notification/internal/infrastructure/http/handler"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/shared/pkg/middleware"
)

// RouterDeps agrupa las dependencias del router de notification.
type RouterDeps struct {
	Logger        *slog.Logger
	ServiceName   string
	HealthHandler *handler.HealthHandler
}

// NewRouter construye el router de notification.
// notification tiene un API HTTP mínima — su trabajo principal
// es el consumo NATS y la entrega de webhooks.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := logger.WithLogger(req.Context(), deps.Logger)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Use(middleware.Recover)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)

	// Health check — único endpoint público de notification
	r.Get("/health", deps.HealthHandler.Health)
	r.Get("/api/v1/health", deps.HealthHandler.Health)

	return r
}
