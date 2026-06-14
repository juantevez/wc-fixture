// Package middleware provee middleware HTTP reutilizables para todos los
// bounded contexts. Cada middleware sigue la firma estándar de chi/net/http.
package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/logger"
)

const headerRequestID = "X-Request-Id"

// RequestID inyecta un request ID único en cada request.
// Si el header X-Request-Id ya viene en el request entrante, lo reutiliza.
// Siempre lo reenvía en la respuesta.
// También lo inyecta en el contexto del logger.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(headerRequestID)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set(headerRequestID, requestID)

		// Enriquecer el logger del contexto con el request ID.
		log := logger.FromContext(r.Context()).With("request_id", requestID)
		r = r.WithContext(logger.WithLogger(r.Context(), log))

		next.ServeHTTP(w, r)
	})
}
