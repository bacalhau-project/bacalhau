package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var NoTLS = LegacyTLSSupport{UseTLS: false}

// APIClient is a utility for interacting with a node's API server against v1 APIs.
type APIClient struct {
	BaseURI        *url.URL
	DefaultHeaders map[string]string
	Client         *http.Client
}

// LegacyTLSSupport contains information on how to use TLS (or not) to communicate
// with the v1 APIs
type LegacyTLSSupport struct {
	UseTLS   bool
	CACert   string
	Insecure bool
}

// NewAPIClient returns a new client for a node's API server against v1 APIs
// the client will use /api/v1 path by default is no custom path is defined
func NewAPIClient(tlsinfo LegacyTLSSupport, host string, port uint16, path ...string) *APIClient {
	scheme := "http"
	if tlsinfo.UseTLS {
		scheme = "https"
	}

	baseURI := system.MustParseURL(fmt.Sprintf("%s://%s:%d", scheme, host, port)).JoinPath(path...)

	tr := getTLSTransport(tlsinfo)

	return &APIClient{
		BaseURI:        baseURI,
		DefaultHeaders: map[string]string{},

		Client: &http.Client{
			Timeout: 300 * time.Second,
			Transport: otelhttp.NewTransport(tr,
				otelhttp.WithSpanOptions(
					trace.WithAttributes(
						attribute.String("clientID", system.GetClientID()),
					),
				),
			),
		},
	}
}

// getTLSTransport builds a http.Transport from the TLS options
func getTLSTransport(config LegacyTLSSupport) *http.Transport {
	tr := &http.Transport{}

	if !config.UseTLS {
		return tr
	}

	if config.CACert != "" {
		caCert, err := os.ReadFile(config.CACert)
		if err != nil {
			// unreachable: we already checked that the file exists at CLI startup
			// if it has gone missing in the meantime then something is very wrong
			newErr := errors.Wrap(err, fmt.Sprintf("Error: unable to read CA certificate: %s", config.CACert))
			panic(newErr.Error())
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tr.TLSClientConfig = &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		}
	} else if config.Insecure {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS12,
		}
	}
	return tr
}

func (apiClient *APIClient) doGet(ctx context.Context, api string, resData any) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Get")
	defer span.End()

	addr := apiClient.BaseURI.JoinPath(api).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error creating Get request: %v", err))
	}
	return apiClient.do(ctx, req, resData)
}

func (apiClient *APIClient) doPostSigned(ctx context.Context, api string, reqData, resData interface{}) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.DoPostSigned")
	defer span.End()

	req, err := signatures.SignRequest(reqData)
	if err != nil {
		return err
	}

	return apiClient.DoPost(ctx, api, req, resData)
}

func (apiClient *APIClient) DoPost(ctx context.Context, api string, reqData, resData interface{}) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.DoPost")
	defer span.End()

	var body bytes.Buffer
	var err error
	if err = json.NewEncoder(&body).Encode(reqData); err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error encoding request body: %v", err))
	}

	addr := apiClient.BaseURI.JoinPath(api).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, &body)
	if err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error creating Post request: %v", err))
	}
	req.Header.Set("Content-type", "application/json")
	return apiClient.do(ctx, req, resData)
}

func (apiClient *APIClient) do(ctx context.Context, req *http.Request, resData any) error {
	for header, value := range apiClient.DefaultHeaders {
		req.Header.Set(header, value)
	}
	req.Close = true // don't keep connections lying around

	var res *http.Response
	res, err := apiClient.Client.Do(req)
	if err != nil {
		errString := err.Error()
		if errorResponse, ok := err.(*bacerrors.ErrorResponse); ok {
			return errorResponse
		} else if errString == "context canceled" {
			return bacerrors.NewContextCanceledError(err.Error())
		} else {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: after posting request: %v", err))
		}
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			err = fmt.Errorf("error closing response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		var responseBody []byte
		responseBody, err = io.ReadAll(res.Body)
		if err != nil {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error reading response body: %v", err))
		}

		var serverError *bacerrors.ErrorResponse
		if err = marshaller.JSONUnmarshalWithMax(responseBody, &serverError); err != nil {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: after posting request: %v",
				string(responseBody)))
		}

		if !reflect.DeepEqual(serverError, bacerrors.BacalhauErrorInterface(nil)) {
			return serverError
		}
	}

	err = json.NewDecoder(res.Body).Decode(resData)
	if err != nil {
		if err == io.EOF {
			return nil // No error, just no data
		} else {
			return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error decoding response body: %v", err))
		}
	}

	return nil
}
