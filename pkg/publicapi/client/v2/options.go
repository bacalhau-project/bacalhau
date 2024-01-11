//nolint:unused
package client

import (
	"context"
	"net/http"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Options struct {
	// Address is the address of the node's public REST API.
	Address string

	// Namespace is the default namespace to use for all requests.
	Namespace string

	// The optional application specific identifier appended to the User-Agent header.
	AppID string

	// Context is the default context to use for requests.
	Context context.Context

	// HTTPClient is the client to use. Default will be used if not provided.
	// If set, other configuration options will be ignored, such as Timeout
	HTTPClient *http.Client

	// HTTPAuth is the auth info to use for http access.
	HTTPAuth *apimodels.HTTPBasicAuth

	// Timeout is the timeout for requests.
	Timeout time.Duration

	// Headers is a map of headers to add to all requests.
	Headers http.Header
}

// OptionFn is a function that can be used to configure the client.
type OptionFn func(*Options)

// WithAddress sets the address of the node's public REST API.
func WithAddress(address string) OptionFn {
	return func(o *Options) {
		o.Address = address
	}
}

// WithNamespace sets the default namespace to use for all requests.
func WithNamespace(namespace string) OptionFn {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// WithAppID sets the optional application specific identifier appended to the User-Agent header.
func WithAppID(appID string) OptionFn {
	return func(o *Options) {
		o.AppID = appID
	}
}

// WithContext sets the default context to use for requests.
func WithContext(ctx context.Context) OptionFn {
	return func(o *Options) {
		o.Context = ctx
	}
}

// WithHTTPClient sets the client to use. Default will be used if not provided.
// If set, other configuration options will be ignored, such as Timeout
func WithHTTPClient(client *http.Client) OptionFn {
	return func(o *Options) {
		o.HTTPClient = client
	}
}

// WithHTTPAuth sets the auth info to use for http access.
func WithHTTPAuth(auth *apimodels.HTTPBasicAuth) OptionFn {
	return func(o *Options) {
		o.HTTPAuth = auth
	}
}

// WithTimeout sets the timeout for requests.
func WithTimeout(timeout time.Duration) OptionFn {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// WithHeaders sets the headers to add to all requests.
func WithHeaders(headers http.Header) OptionFn {
	return func(o *Options) {
		o.Headers = headers
	}
}

func resolveHTTPClient(config *Options) {
	if config.HTTPClient != nil {
		return
	}
	config.HTTPClient = defaultHTTPClient(config)
}

// defaultHTTPClient is the default client to use if none is provided.
func defaultHTTPClient(config *Options) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
		Transport: otelhttp.NewTransport(nil,
			otelhttp.WithSpanOptions(
				trace.WithAttributes(
					attribute.String("AppID", config.AppID),
				),
			),
		),
	}
}
