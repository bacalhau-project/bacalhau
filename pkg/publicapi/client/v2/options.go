//nolint:unused
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
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

	// TLSOptions provides info on how we want to use TLS
	TLS TLSOptions
}

type TLSOptions struct {
	// UseTLS denotes whether to use TLS or not
	UseTLS bool
	// Insecure activates TLS but does not verify any certificate
	Insecure bool
	// CACert specifies the location of a self-signed CA certificate
	// file
	CACert string
}

// OptionFn is a function that can be used to configure the client.
type OptionFn func(*Options)

// UseTLS denotes whether to use TLS or not
func WithTLS(active bool) OptionFn {
	return func(o *Options) {
		o.TLS.UseTLS = active
	}
}

// CACert specifies the location of a CA certificate file so
// that it is possible to use TLS without the insecure flag
// when the server uses a self-signed certificate
func WithCACertificate(cacert string) OptionFn {
	return func(o *Options) {
		o.TLS.CACert = cacert
	}
}

// Insecure activates TLS but does not verify any certificate
func WithInsecureTLS(insecure bool) OptionFn {
	return func(o *Options) {
		o.TLS.Insecure = insecure
	}
}

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

// getTLSTransport builds a http.Transport from the TLS options
func getTLSTransport(config *Options) *http.Transport {
	tr := &http.Transport{}

	if !config.TLS.UseTLS {
		return tr
	}

	if config.TLS.CACert != "" {
		caCert, err := os.ReadFile(config.TLS.CACert)
		if err != nil {
			panic("invalid ca certificate provided")
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tr.TLSClientConfig = &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		}
	} else if config.TLS.Insecure {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS12,
		}
	}
	return tr
}

// defaultHTTPClient is the default client to use if none is provided.
func defaultHTTPClient(config *Options) *http.Client {
	tr := getTLSTransport(config)

	return &http.Client{
		Timeout: config.Timeout,
		Transport: otelhttp.NewTransport(tr,
			otelhttp.WithSpanOptions(
				trace.WithAttributes(
					attribute.String("AppID", config.AppID),
				),
			),
		),
	}
}
