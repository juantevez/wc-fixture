// Package httputil provee helpers de respuesta HTTP compartidos entre todos
// los bounded contexts. Estandariza el formato JSON de respuestas exitosas
// y de error, garantizando consistencia en la API pública.
package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// envelope es el wrapper estándar para respuestas exitosas.
//
//	{ "data": <payload> }
type envelope[T any] struct {
	Data T `json:"data"`
}

// errorBody es el formato estándar de error para el cliente.
//
//	{ "error": { "code": "NOT_FOUND", "message": "..." } }
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON serializa data como JSON con el status dado.
// Siempre envuelve en { "data": ... } para extensibilidad futura.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope[any]{Data: data})
}

// WriteOK es un alias de WriteJSON con status 200.
func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, data)
}

// WriteCreated es un alias de WriteJSON con status 201.
func WriteCreated(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusCreated, data)
}

// WriteNoContent escribe una respuesta 204 sin cuerpo.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError escribe una respuesta de error en formato estándar.
// Si err es un *apperrors.AppError, usa su código y mensaje.
// Si es otro tipo de error, responde 500 sin exponer el detalle interno.
func WriteError(ctx interface{ Value(any) any }, w http.ResponseWriter, r *http.Request, err error) {
	var ae *apperrors.AppError
	if errors.As(err, &ae) {
		body := errorBody{Error: errorDetail{
			Code:    string(ae.Code),
			Message: ae.Message,
		}}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ae.Code.HTTPStatus())
		_ = json.NewEncoder(w).Encode(body)
		return
	}

	// Error interno no tipado: loguear y responder 500 genérico.
	log := logger.FromContext(r.Context())
	log.Error("error interno no manejado", slog.String("error", err.Error()), slog.String("path", r.URL.Path))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorDetail{
		Code:    string(apperrors.CodeInternal),
		Message: "error interno del servidor",
	}})
}
