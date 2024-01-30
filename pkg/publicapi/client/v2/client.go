package client

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

type Client struct {
	address string

	httpClient *http.Client
	config     Config
}

// New creates a new client.
func New(address string, optFns ...OptionFn) *Client {
	// define default filed on the config by setting them here, then
	// modify with options to override.
	var cfg Config
	for _, optFn := range optFns {
		optFn(&cfg)
	}

	resolveHTTPClient(&cfg)
	return &Client{
		address:    address,
		httpClient: cfg.HTTPClient,
		config:     cfg,
	}
}

// get is used to do a GET request against an endpoint
// and deserialize the response into a response object
func (c *Client) get(ctx context.Context, endpoint string, in apimodels.GetRequest, out apimodels.GetResponse) error {
	r := in.ToHTTPRequest()
	_, resp, err := requireOK(c.doRequest(ctx, http.MethodGet, endpoint, r)) //nolint:bodyclose // this is being closed
	if err != nil {
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
func (c *Client) write(ctx context.Context, verb, endpoint string, in apimodels.PutRequest,
	out apimodels.Response) error {
	r := in.ToHTTPRequest()
	if r.BodyObj == nil && r.Body == nil {
		r.BodyObj = in
	}
	_, resp, err := requireOK(c.doRequest(ctx, verb, endpoint, r)) //nolint:bodyclose // this is being closed
	if err != nil {
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

// list is used to do a GET request against an endpoint
// and deserialize the response into a response object
func (c *Client) list(ctx context.Context, endpoint string, in apimodels.ListRequest,
	out apimodels.ListResponse) error {
	return c.get(ctx, endpoint, in, out)
}

// put is used to do a PUT request against an endpoint
func (c *Client) put(ctx context.Context, endpoint string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return c.write(ctx, http.MethodPut, endpoint, in, out)
}

// post is used to do a POST request against an endpoint
//
//nolint:unused
func (c *Client) post(ctx context.Context, endpoint string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return c.write(ctx, http.MethodPost, endpoint, in, out)
}

// delete is used to do a DELETE request against an endpoint
func (c *Client) delete(ctx context.Context, endpoint string, in apimodels.PutRequest, out apimodels.Response) error {
	return c.write(ctx, http.MethodDelete, endpoint, in, out)
}

// doRequest runs a request with our client
func (c *Client) doRequest(ctx context.Context, method, endpoint string, r *apimodels.HTTPRequest) (time.Duration, *http.Response, error) {
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
func (c *Client) toHTTP(ctx context.Context, method, endpoint string, r *apimodels.HTTPRequest) (*http.Request, error) {
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

	// Optionally configure HTTP authorization
	if c.config.HTTPAuth != nil {
		req.Header.Set("Authorization", c.config.HTTPAuth.String())
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
func (c *Client) url(endpoint string) (*url.URL, error) {
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
