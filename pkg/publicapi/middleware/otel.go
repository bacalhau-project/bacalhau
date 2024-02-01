package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

func Otel() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			w := c.Response().Writer

			handler := otelhttp.NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					routePattern := c.Path() // Get the route pattern from Echo's context

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

			handler.ServeHTTP(w, r)
			return next(c) // Call the next handler in the chain after the telemetry has been set up
		}
	}
}
