package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wc-fixture/result-ingestion/internal/infrastructure/http/handler"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/shared/pkg/middleware"
)

// RouterDeps agrupa las dependencias del router.
type RouterDeps struct {
	Logger        *slog.Logger
	ServiceName   string
	ResultHandler *handler.ResultHandler
	InternalToken string // token para proteger el endpoint de ingesta
}

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
	r.Use(middleware.Tracing(deps.ServiceName))
	r.Use(middleware.Logging)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"result-ingestion"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Endpoint protegido con token interno
		r.With(internalTokenAuth(deps.InternalToken)).
			Post("/results", deps.ResultHandler.IngestResult)
	})

	return r
}

// internalTokenAuth es un middleware que verifica el header X-Internal-Token.
func internalTokenAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token != "" && r.Header.Get("X-Internal-Token") != token {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"token interno requerido"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
