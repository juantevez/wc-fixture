// Package middleware contiene middleware HTTP específicos de fixture-core.
// Los middleware transversales (logging, tracing, recover, requestid) viven
// en shared/pkg/middleware y se aplican en el router.
package middleware

import (
	"net/http"
	"strings"

	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
)

// InternalOnly restringe un endpoint a llamadas internas del cluster.
// Verifica el header X-Internal-Token contra el valor configurado.
// Se usa para endpoints de escritura (POST /result, PUT /schedule)
// que solo deben ser accesibles desde result-ingestion o admin.
func InternalOnly(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			incoming := r.Header.Get("X-Internal-Token")
			if incoming == "" || incoming != token {
				err := apperrors.Unauthorized("acceso denegado: se requiere token interno")
				httputil.WriteError(r.Context(), w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BearerToken extrae el Bearer token del header Authorization.
// Retorna "" si el header no está presente o tiene formato incorrecto.
func BearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}
