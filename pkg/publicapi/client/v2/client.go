package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

// Client is the object that makes transport-level requests to specified APIs.
// Users should make use of the `API` object for a higher level interface.
type Client interface {
	Get(context.Context, string, apimodels.GetRequest, apimodels.GetResponse) error
	List(context.Context, string, apimodels.ListRequest, apimodels.ListResponse) error
	Put(context.Context, string, apimodels.PutRequest, apimodels.PutResponse) error
	Post(context.Context, string, apimodels.PutRequest, apimodels.PutResponse) error
	Delete(context.Context, string, apimodels.PutRequest, apimodels.Response) error
	Dial(context.Context, string, apimodels.Request) (<-chan *concurrency.AsyncResult[[]byte], error)
}

// New creates a new transport.
func NewHTTPClient(address string, optFns ...OptionFn) Client {
	// define default filed on the config by setting them here, then
	// modify with options to override.
	var cfg Config
	for _, optFn := range optFns {
		optFn(&cfg)
	}

	resolveHTTPClient(&cfg)
	return &httpClient{
		address:    address,
		httpClient: cfg.HTTPClient,
		config:     cfg,
	}
}

type httpClient struct {
	address string

	httpClient *http.Client
	config     Config
}

// Get is used to do a GET request against an endpoint
// and deserialize the response into a response object
func (c *httpClient) Get(ctx context.Context, endpoint string, in apimodels.GetRequest, out apimodels.GetResponse) error {
	r := in.ToHTTPRequest()
	_, resp, err := requireOK(c.doRequest(ctx, http.MethodGet, endpoint, r)) //nolint:bodyclose // this is being closed
	if err != nil && resp != nil && resp.StatusCode == http.StatusUnauthorized {
		return apimodels.ErrInvalidToken
	} else if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out != nil {
		if err := decodeBody(resp, &out); err != nil {
			return err
		}
		out.Normalize()
	}
	return nil
}

// write is used to do a write request against an endpoint
// You probably want the delete, post, or put methods.
func (c *httpClient) write(ctx context.Context, verb, endpoint string, in apimodels.PutRequest,
	out apimodels.Response) error {
	r := in.ToHTTPRequest()
	if r.BodyObj == nil && r.Body == nil {
		r.BodyObj = in
	}
	_, resp, err := requireOK(c.doRequest(ctx, verb, endpoint, r)) //nolint:bodyclose // this is being closed
	if err != nil && resp != nil && resp.StatusCode == http.StatusUnauthorized {
		return apimodels.ErrInvalidToken
	} else if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out != nil {
		if err := decodeBody(resp, &out); err != nil {
			return err
		}
		out.Normalize()
	}
	return nil
}

// List is used to do a GET request against an endpoint
// and deserialize the response into a response object
func (c *httpClient) List(ctx context.Context, endpoint string, in apimodels.ListRequest, out apimodels.ListResponse) error {
	return c.Get(ctx, endpoint, in, out)
}

// Put is used to do a PUT request against an endpoint
func (c *httpClient) Put(ctx context.Context, endpoint string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return c.write(ctx, http.MethodPut, endpoint, in, out)
}

// Post is used to do a POST request against an endpoint
func (c *httpClient) Post(ctx context.Context, endpoint string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return c.write(ctx, http.MethodPost, endpoint, in, out)
}

// Delete is used to do a DELETE request against an endpoint
func (c *httpClient) Delete(ctx context.Context, endpoint string, in apimodels.PutRequest, out apimodels.Response) error {
	return c.write(ctx, http.MethodDelete, endpoint, in, out)
}

// Dial is used to upgrade to a Websocket connection and subscribe to an
// endpoint. The method returns on error or if the endpoint has been
// successfully dialed, from which point on the returned channel will contain
// every received message.
func (c *httpClient) Dial(ctx context.Context, endpoint string, in apimodels.Request) (<-chan *concurrency.AsyncResult[[]byte], error) {
	r := in.ToHTTPRequest()
	httpR, err := c.toHTTP(ctx, http.MethodGet, endpoint, r)
	if err != nil {
		return nil, err
	}

	dialer := *websocket.DefaultDialer
	httpR.URL.Scheme = "ws"

	// if we are using TLS create a TLS config
	if c.config.TLS.UseTLS {
		httpR.URL.Scheme = "wss"
		dialer.TLSClientConfig = getTLSTransport(&c.config).TLSClientConfig
	}

	// Connect to the server
	conn, resp, err := dialer.Dial(httpR.URL.String(), httpR.Header)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read messages from the server, and send them until the conn is closed or
	// the context is cancelled. We have to read them here because the reader
	// will be discarded upon the next call to NextReader.
	output := make(chan *concurrency.AsyncResult[[]byte], c.config.WebsocketChannelBuffer)
	go func() {
		defer func() {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			close(output)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, reader, err := conn.NextReader()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
						output <- &concurrency.AsyncResult[[]byte]{Err: err}
					}
					return
				}

				if reader != nil {
					var buf bytes.Buffer
					if _, err := io.Copy(&buf, reader); err != nil {
						output <- &concurrency.AsyncResult[[]byte]{Err: err}
						return
					}
					output <- &concurrency.AsyncResult[[]byte]{Value: buf.Bytes()}
				}
			}
		}
	}()

	return output, nil
}

// doRequest runs a request with our client
func (c *httpClient) doRequest(
	ctx context.Context,
	method, endpoint string,
	r *apimodels.HTTPRequest,
) (time.Duration, *http.Response, error) {
	req, err := c.toHTTP(ctx, method, endpoint, r)
	if err != nil {
		return 0, nil, err
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	diff := time.Since(start)

	// If the response is compressed, we swap the body's reader.
	if zipErr := autoUnzip(resp); zipErr != nil {
		return 0, nil, zipErr
	}

	return diff, resp, err
}

// toHTTP converts the request to an HTTP request
func (c *httpClient) toHTTP(ctx context.Context, method, endpoint string, r *apimodels.HTTPRequest) (*http.Request, error) {
	u, err := c.url(endpoint)
	if err != nil {
		return nil, err
	}

	// build parameters
	if c.config.Namespace != "" && r.Params.Get("namespace") == "" {
		r.Params.Add("namespace", c.config.Namespace)
	}
	// Add in the query parameters, if any
	for key, values := range u.Query() {
		for _, value := range values {
			r.Params.Add(key, value)
		}
	}
	// Encode the query parameters
	u.RawQuery = r.Params.Encode()

	// Check if we should encode the body
	contentType := ""
	body := r.Body
	if body == nil && r.BodyObj != nil {
		if body, err = encodeBody(r.BodyObj); err != nil {
			return nil, err
		}
		contentType = "application/json"
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, method, u.RequestURI(), body)
	if err != nil {
		return nil, err
	}

	// build headers
	req.Header = r.Header
	req.Header.Add("Accept-Encoding", "gzip")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.config.AppID != "" {
		req.Header.Set(apimodels.HTTPHeaderAppID, c.config.AppID)
		req.Header.Add("User-Agent", c.config.AppID)
	}

	for key, values := range c.config.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.URL.Host = u.Host
	req.URL.Scheme = u.Scheme
	req.Host = u.Host
	return req, nil
}

// generate URL for a given endpoint
func (c *httpClient) url(endpoint string) (*url.URL, error) {
	base, err := url.Parse(c.address)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Scheme:  base.Scheme,
		User:    base.User,
		Host:    base.Host,
		Path:    u.Path,
		RawPath: u.RawPath,
	}, nil
}

// AuthenticatingClient is a client implementation that will automatically run
// user authentication when a new authorization token is required. This is
// either when the user does not yet have an authorization token that matches
// the remote server or if the token is used but the server says it is invalid
// (e.g. because it has expired).
//
// Since authentication is normally an interactive affair, this client requires
// an authentication callback that will be called to actually authenticate.
//
// Authorization tokens can be optionally persisted by supplying a callback.
// This client will keep track of any authorization tokens it collects.
type AuthenticatingClient struct {
	Client Client

	// Credential should be any existing client credential for the user. It is
	// allowed to be nil, representing no existing client credential.
	Credential *apimodels.HTTPCredential

	// PersistCredential will be called when the system should remember a new
	// auth token for a user. The supplied auth token may be nil, in which case
	// any existing tokens should be deleted.
	PersistCredential func(*apimodels.HTTPCredential) error

	// Authenticate will be called when the system should run an authentication
	// flow using the passed Auth API.
	Authenticate func(context.Context, *Auth) (*apimodels.HTTPCredential, error)
}

func (t *AuthenticatingClient) Get(ctx context.Context, path string, in apimodels.GetRequest, out apimodels.GetResponse) error {
	return doRequest(ctx, t, in, func(req apimodels.GetRequest) error {
		return t.Client.Get(ctx, path, req, out)
	})
}

func (t *AuthenticatingClient) List(ctx context.Context, path string, in apimodels.ListRequest, out apimodels.ListResponse) error {
	return doRequest(ctx, t, in, func(req apimodels.ListRequest) error {
		return t.Client.List(ctx, path, req, out)
	})
}

func (t *AuthenticatingClient) Post(ctx context.Context, path string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return doRequest(ctx, t, in, func(req apimodels.PutRequest) error {
		return t.Client.Post(ctx, path, req, out)
	})
}

func (t *AuthenticatingClient) Put(ctx context.Context, path string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return doRequest(ctx, t, in, func(req apimodels.PutRequest) error {
		return t.Client.Put(ctx, path, req, out)
	})
}

func (t *AuthenticatingClient) Delete(ctx context.Context, path string, in apimodels.PutRequest, out apimodels.Response) error {
	return doRequest(ctx, t, in, func(req apimodels.PutRequest) error {
		return t.Client.Delete(ctx, path, req, out)
	})
}

func (t *AuthenticatingClient) Dial(
	ctx context.Context,
	path string,
	in apimodels.Request,
) (<-chan *concurrency.AsyncResult[[]byte], error) {
	var output <-chan *concurrency.AsyncResult[[]byte]
	err := doRequest(ctx, t, in, func(req apimodels.Request) (err error) {
		output, err = t.Client.Dial(ctx, path, req)
		return
	})
	return output, err
}

func doRequest[R apimodels.Request](ctx context.Context, t *AuthenticatingClient, request R, runRequest func(R) error) (err error) {
	if t.Credential != nil {
		request.SetCredential(t.Credential)
		if err = runRequest(request); err == nil {
			// Initial request with auth token was successful.
			return nil
		} else if t.Authenticate == nil {
			// We don't have an authenticate method so can't try and get a new
			// token, so we need to stop here.
			return pkgerrors.Wrap(err, "unauthorized and no authentication is available")
		}
	}

	// If we don't have a credential yet or the token we had was invalid, run a
	// new auth flow to get a new token (maybe).
	if t.Credential == nil || pkgerrors.Is(err, apimodels.ErrInvalidToken) {
		var authErr error
		auth := NewAPI(t.Client).Auth()
		if t.Credential, err = t.Authenticate(ctx, auth); err != nil {
			authErr = errors.Join(authErr, pkgerrors.Wrap(err, "failed to authorize user"))
			t.Credential = nil // Don't assume Authenticate returned nil
		}

		// We either failed to get a credential or have a new one. Either way,
		// persist the result of the call to remove the old credential.
		if err = t.PersistCredential(t.Credential); err != nil {
			authErr = errors.Join(authErr, pkgerrors.Wrap(err, "unable to persist new client credential"))
		}
		err = authErr
	}

	if err != nil {
		// Initial request unsuccessful, but not due to invalid/missing token,
		// or we failed to authenticate/persist. Either way, return the error.
		return err
	}

	// Try the initial request again with our possible new credential. It's ok
	// if we didn't authenticate because this server might accept
	// unauthenticated requests.
	request.SetCredential(t.Credential)
	return runRequest(request)
}

var _ Client = (*AuthenticatingClient)(nil)
