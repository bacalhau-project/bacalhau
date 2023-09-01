package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func Otel(next http.Handler) http.Handler {
	return otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			routePattern := chi.RouteContext(r.Context()).RoutePattern()

			span := trace.SpanFromContext(r.Context())
			span.SetName(routePattern)
			span.SetAttributes(semconv.HTTPTarget(r.URL.String()), semconv.HTTPRoute(routePattern))

			labeler, ok := otelhttp.LabelerFromContext(r.Context())
			if ok {
				labeler.Add(semconv.HTTPRoute(routePattern))
			}
		}),
		"",
		otelhttp.WithPublicEndpoint(),
	)
}
