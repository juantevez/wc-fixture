// Package handler contiene los handlers HTTP del bounded context notification.
package handler

import (
	"net/http"
)

// HealthHandler maneja el endpoint de health check.
// notification solo expone este endpoint HTTP público —
// su trabajo real lo hace vía NATS consumer y webhooks salientes.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health responde 200 OK con estado del servicio.
//
//	GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"notification"}`))
}
