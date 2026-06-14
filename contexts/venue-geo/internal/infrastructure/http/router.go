package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/shared/pkg/middleware"
	"github.com/wc-fixture/venue-geo/internal/infrastructure/http/handler"
)

// RouterDeps agrupa las dependencias del router.
type RouterDeps struct {
	Logger       *slog.Logger
	ServiceName  string
	VenueHandler *handler.VenueHandler
}

// NewRouter construye el router chi con middlewares y rutas de venue-geo.
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

	// Health check
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"venue-geo"}`))
	})

	// API v1 — venue-geo expone endpoints de solo lectura
	r.Route("/api/v1/venues", func(r chi.Router) {
		r.Get("/", deps.VenueHandler.ListVenues)                  // GET /venues?country=USA
		r.Get("/distance", deps.VenueHandler.GetDistance)          // GET /venues/distance?from=&to=
		r.Get("/distance-matrix", deps.VenueHandler.GetDistanceMatrix) // GET /venues/distance-matrix
		r.Get("/nearby", deps.VenueHandler.GetNearbyVenues)        // GET /venues/nearby?lat=&lon=&radius_km=
		r.Get("/{venueID}", deps.VenueHandler.GetVenue)            // GET /venues/{id}
	})

	return r
}
