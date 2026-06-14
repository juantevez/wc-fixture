package middleware

import (
	"net/http"

	"github.com/wc-fixture/shared/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// Tracing extrae el contexto de tracing del header entrante (W3C TraceContext),
// inicia un span para el request y lo propaga en el contexto.
// También enriquece el logger con trace_id y span_id para correlación de logs.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extraer contexto de tracing del header W3C (traceparent, tracestate).
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
			defer span.End()

			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.route", r.URL.Path),
			)

			// Inyectar trace_id en el logger del contexto para correlación.
			spanCtx := span.SpanContext()
			if spanCtx.IsValid() {
				log := logger.FromContext(ctx).With(
					"trace_id", spanCtx.TraceID().String(),
					"span_id", spanCtx.SpanID().String(),
				)
				ctx = logger.WithLogger(ctx, log)
			}

			rw := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r.WithContext(ctx))

			span.SetAttributes(attribute.Int("http.status_code", rw.Status()))
		})
	}
}
