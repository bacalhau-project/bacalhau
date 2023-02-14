package publicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// APIClient is a utility for interacting with a node's API server.
type APIClient struct {
	BaseURI        string
	DefaultHeaders map[string]string

	Client *http.Client
}

// NewAPIClient returns a new client for a node's API server.
func NewAPIClient(baseURI string) *APIClient {
	return &APIClient{
		BaseURI:        baseURI,
		DefaultHeaders: map[string]string{},

		Client: &http.Client{
			Timeout: 300 * time.Second,
			Transport: otelhttp.NewTransport(nil,
				otelhttp.WithSpanOptions(
					trace.WithAttributes(
						attribute.String("clientID", system.GetClientID()),
					),
				),
			),
		},
	}
}

// Alive calls the node's API server health check.
func (apiClient *APIClient) Alive(ctx context.Context) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Alive")
	defer span.End()

	var body io.Reader
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiClient.BaseURI+"/livez", body)
	if err != nil {
		return false, nil
	}
	res, err := apiClient.Client.Do(req) //nolint:bodyclose // golangcilint is dumb - this is closed
	if err != nil {
		return false, nil
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "apiClient response", res.Body)

	return res.StatusCode == http.StatusOK, nil
}

func (apiClient *APIClient) Version(ctx context.Context) (*model.BuildVersionInfo, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Version")
	defer span.End()

	req := VersionRequest{
		ClientID: system.GetClientID(),
	}

	var res VersionResponse
	if err := apiClient.Post(ctx, "version", req, &res); err != nil {
		return nil, err
	}

	return res.VersionInfo, nil
}

func (apiClient *APIClient) Post(ctx context.Context, api string, reqData, resData interface{}) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Post")
	defer span.End()

	var body bytes.Buffer
	var err error
	if err = json.NewEncoder(&body).Encode(reqData); err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error encoding request body: %v", err))
	}

	addr := fmt.Sprintf("%s/%s", apiClient.BaseURI, api)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, &body)
	if err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error creating Post request: %v", err))
	}
	req.Header.Set("Content-type", "application/json")
	for header, value := range apiClient.DefaultHeaders {
		req.Header.Set(header, value)
	}
	req.Close = true // don't keep connections lying around

	var res *http.Response
	res, err = apiClient.Client.Do(req)
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
		if err = model.JSONUnmarshalWithMax(responseBody, &serverError); err != nil {
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
