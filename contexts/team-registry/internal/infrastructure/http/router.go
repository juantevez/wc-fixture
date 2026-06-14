package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/shared/pkg/middleware"
	"github.com/wc-fixture/team-registry/internal/infrastructure/http/handler"
)

// RouterDeps agrupa las dependencias del router.
type RouterDeps struct {
	Logger      *slog.Logger
	ServiceName string
	TeamHandler *handler.TeamHandler
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
		_, _ = w.Write([]byte(`{"status":"ok","service":"team-registry"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Equipos
		r.Get("/teams", deps.TeamHandler.ListTeams)
		r.Get("/teams/{teamID}", deps.TeamHandler.GetTeam)

		// Confederaciones
		r.Get("/confederations", deps.TeamHandler.ListConfederations)
		r.Get("/confederations/{code}", deps.TeamHandler.GetConfederation)
	})

	return r
}
