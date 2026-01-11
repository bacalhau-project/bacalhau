package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	// Namespace is the default namespace to use for all requests.
	Namespace string

	// The optional application specific identifier appended to the User-Agent header.
	AppID string

	// HTTPClient is the client to use. Default will be used if not provided.
	// If set, other configuration options will be ignored, such as Timeout
	HTTPClient *http.Client

	// Timeout is the timeout for requests.
	Timeout time.Duration

	// Headers is a map of headers to add to all requests.
	Headers http.Header

	// TLSConfig provides info on how we want to use TLS
	TLS TLSConfig

	// WebsocketChannelBuffer is the size of the channel buffer for websocket messages
	WebsocketChannelBuffer int
}

// DefaultConfig returns a default configuration for the client.
func DefaultConfig() Config {
	return Config{
		Timeout:                30 * time.Second,
		WebsocketChannelBuffer: 10,
	}
}

type TLSConfig struct {
	// UseTLS denotes whether to use TLS or not
	UseTLS bool
	// Insecure activates TLS but does not verify any certificate
	Insecure bool
	// CACert specifies the location of a self-signed CA certificate
	// file
	CACert string
}

// OptionFn is a function that can be used to configure the client.
type OptionFn func(*Config)

// UseTLS denotes whether to use TLS or not
func WithTLS(active bool) OptionFn {
	return func(o *Config) {
		o.TLS.UseTLS = active
	}
}

// CACert specifies the location of a CA certificate file so
// that it is possible to use TLS without the insecure flag
// when the server uses a self-signed certificate
func WithCACertificate(cacert string) OptionFn {
	return func(o *Config) {
		o.TLS.CACert = cacert
	}
}

// Insecure activates TLS but does not verify any certificate
func WithInsecureTLS(insecure bool) OptionFn {
	return func(o *Config) {
		o.TLS.Insecure = insecure
	}
}

// WithNamespace sets the default namespace to use for all requests.
func WithNamespace(namespace string) OptionFn {
	return func(o *Config) {
		o.Namespace = namespace
	}
}

// WithAppID sets the optional application specific identifier appended to the User-Agent header.
func WithAppID(appID string) OptionFn {
	return func(o *Config) {
		o.AppID = appID
	}
}

// WithHTTPClient sets the client to use. Default will be used if not provided.
// If set, other configuration options will be ignored, such as Timeout
func WithHTTPClient(client *http.Client) OptionFn {
	return func(o *Config) {
		o.HTTPClient = client
	}
}

// WithTimeout sets the timeout for requests.
func WithTimeout(timeout time.Duration) OptionFn {
	return func(o *Config) {
		o.Timeout = timeout
	}
}

// WithHeaders sets the headers to add to all requests.
func WithHeaders(headers http.Header) OptionFn {
	return func(o *Config) {
		o.Headers = headers
	}
}

// WithWebsocketChannelBuffer sets the size of the channel buffer for websocket messages
func WithWebsocketChannelBuffer(buffer int) OptionFn {
	return func(o *Config) {
		o.WebsocketChannelBuffer = buffer
	}
}

func resolveHTTPClient(config *Config) {
	if config.HTTPClient != nil {
		return
	}
	config.HTTPClient = defaultHTTPClient(config)
}

// getTLSTransport builds a http.Transport from the TLS options
func getTLSTransport(config *Config) *http.Transport {
	tr := &http.Transport{}

	if !config.TLS.UseTLS {
		return tr
	}

	if config.TLS.CACert != "" {
		caCert, err := os.ReadFile(config.TLS.CACert)
		if err != nil {
			// unreachable: we already checked that the file exists at CLI startup
			// if it has gone missing in the meantime then something is very wrong
			newErr := errors.Wrap(err, fmt.Sprintf("Error: unable to read CA certificate: %s", config.TLS.CACert))
			panic(newErr.Error())
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
func defaultHTTPClient(config *Config) *http.Client {
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
