package publicapi

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// APIClient is a utility for interacting with a node's API server against v1 APIs.
type APIClient struct {
	BaseURI        *url.URL
	DefaultHeaders map[string]string

	Client *http.Client
}

// NewAPIClient returns a new client for a node's API server against v1 APIs
// the client will use /api/v1 path by default is no custom path is defined
func NewAPIClient(host string, port uint16, path ...string) *APIClient {
	baseURI := system.MustParseURL(fmt.Sprintf("http://%s:%d", host, port)).JoinPath(path...)
	if len(path) == 0 {
		baseURI = baseURI.JoinPath(V1APIPrefix)
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiClient.BaseURI.JoinPath("livez").String(), body)
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

func (apiClient *APIClient) Version(ctx context.Context) (*models.BuildVersionInfo, error) {
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

func (apiClient *APIClient) PublicKey(ctx context.Context) (*rsa.PublicKey, string, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.PublicKey")
	defer span.End()

	base := apiClient.BaseURI
	jwksURL := fmt.Sprintf("%s://%s/.well-known/jwks.json", base.Scheme, base.Host)

	set, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		return nil, "", err
	}

	// Get the single key, although we'd ideally do this via the key id.
	key, found := set.Key(0)
	if !found {
		return nil, "", errors.New("could not find key in keyset")
	}

	var rsaKey rsa.PublicKey
	err = key.Raw(&rsaKey)
	if err != nil {
		return nil, "", err
	}

	return &rsaKey, key.KeyID(), nil
}

func (apiClient *APIClient) Get(ctx context.Context, api string, resData any) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Get")
	defer span.End()

	addr := apiClient.BaseURI.JoinPath(api).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return bacerrors.NewResponseUnknownError(fmt.Errorf("publicapi: error creating Get request: %v", err))
	}
	return apiClient.Do(ctx, req, resData)
}

func (apiClient *APIClient) PostSigned(ctx context.Context, api string, reqData, resData interface{}) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.PostSigned")
	defer span.End()

	req, err := SignRequest(reqData)
	if err != nil {
		return err
	}

	return apiClient.Post(ctx, api, req, resData)
}

func (apiClient *APIClient) Post(ctx context.Context, api string, reqData, resData interface{}) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/publicapi.Client.Post")
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
	return apiClient.Do(ctx, req, resData)
}

func (apiClient *APIClient) Do(ctx context.Context, req *http.Request, resData any) error {
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
