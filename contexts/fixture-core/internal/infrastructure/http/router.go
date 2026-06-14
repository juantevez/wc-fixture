package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/wc-fixture/fixture-core/internal/infrastructure/http/handler"
	"github.com/wc-fixture/shared/pkg/middleware"
	"github.com/wc-fixture/shared/pkg/logger"
	"log/slog"
)

// RouterDeps agrupa todas las dependencias necesarias para construir el router.
// Se inyectan desde cmd/server/wire.go.
type RouterDeps struct {
	Logger          *slog.Logger
	ServiceName     string
	FixtureHandler  *handler.FixtureHandler
	GroupHandler    *handler.GroupHandler
	MatchHandler    *handler.MatchHandler
	KnockoutHandler *handler.KnockoutHandler
}

// NewRouter construye el router chi con todos los middlewares y rutas.
// Orden de middlewares (de afuera hacia adentro):
//  1. Recover       — captura panics, siempre primero
//  2. RequestID     — inyecta X-Request-Id en contexto y response
//  3. Tracing       — span OTEL, enriquece logger con trace_id
//  4. Logging       — loguea método/path/status/duración
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Inyectar logger base en el contexto de cada request
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := logger.WithLogger(req.Context(), deps.Logger)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	// Middlewares transversales
	r.Use(middleware.Recover)
	r.Use(middleware.RequestID)
	r.Use(middleware.Tracing(deps.ServiceName))
	r.Use(middleware.Logging)

	// CORS básico para desarrollo — en producción usar un proxy
	r.Use(chimiddleware.SetHeader("Access-Control-Allow-Origin", "*"))

	// Health check — sin autenticación, sin logging verbose
	r.Get("/health", healthHandler)
	r.Get("/api/v1/health", healthHandler)

	// API v1
	r.Route("/api/v1/tournaments/{tournamentID}", func(r chi.Router) {

		// Fixture completo
		r.Get("/fixture", deps.FixtureHandler.GetFixture)

		// Grupos
		r.Get("/groups", deps.GroupHandler.ListGroups)
		r.Get("/groups/{groupName}", deps.GroupHandler.GetGroup)
		r.Get("/groups/{groupName}/standings", deps.GroupHandler.GetStandings)

		// Partidos
		r.Get("/matches", deps.MatchHandler.ListMatches)
		r.Get("/matches/{matchID}", deps.MatchHandler.GetMatch)
		r.Post("/matches/{matchID}/result", deps.MatchHandler.RegisterResult)
		r.Put("/matches/{matchID}/schedule", deps.MatchHandler.UpdateSchedule)

		// Bracket eliminatorio
		r.Get("/knockout", deps.KnockoutHandler.GetKnockout)
		r.Get("/knockout/{phase}", deps.KnockoutHandler.GetKnockoutRound)

		// Mejores terceros
		r.Get("/best-thirds", deps.GroupHandler.GetBestThirds)
	})

	return r
}

// healthHandler responde 200 OK con un JSON mínimo.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
