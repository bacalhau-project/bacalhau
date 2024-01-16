package client

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

type Client struct {
	httpClient *http.Client
	options    Options
}

// New creates a new client.
func New(options Options, optFns ...OptionFn) *Client {
	for _, optFn := range optFns {
		optFn(&options)
	}

	resolveHTTPClient(&options)
	return &Client{
		httpClient: options.HTTPClient,
		options:    options,
	}
}

// get is used to do a GET request against an endpoint
// and deserialize the response into a response object
func (c *Client) get(endpoint string, in apimodels.GetRequest, out apimodels.GetResponse) error {
	r := in.ToHTTPRequest()
	_, resp, err := requireOK(c.doRequest(http.MethodGet, endpoint, r)) //nolint:bodyclose // this is being closed
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
func (c *Client) write(verb, endpoint string, in apimodels.PutRequest, out apimodels.Response) error {
	r := in.ToHTTPRequest()
	if r.BodyObj == nil && r.Body == nil {
		r.BodyObj = in
	}
	_, resp, err := requireOK(c.doRequest(verb, endpoint, r)) //nolint:bodyclose // this is being closed
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
func (c *Client) list(endpoint string, in apimodels.ListRequest, out apimodels.ListResponse) error {
	return c.get(endpoint, in, out)
}

// put is used to do a PUT request against an endpoint
func (c *Client) put(endpoint string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return c.write(http.MethodPut, endpoint, in, out)
}

// post is used to do a POST request against an endpoint
//
//nolint:unused
func (c *Client) post(endpoint string, in apimodels.PutRequest, out apimodels.PutResponse) error {
	return c.write(http.MethodPost, endpoint, in, out)
}

// delete is used to do a DELETE request against an endpoint
func (c *Client) delete(endpoint string, in apimodels.PutRequest, out apimodels.Response) error {
	return c.write(http.MethodDelete, endpoint, in, out)
}

// doRequest runs a request with our client
func (c *Client) doRequest(method, endpoint string, r *apimodels.HTTPRequest) (time.Duration, *http.Response, error) {
	req, err := c.toHTTP(method, endpoint, r)
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
func (c *Client) toHTTP(method, endpoint string, r *apimodels.HTTPRequest) (*http.Request, error) {
	u, err := c.url(endpoint)
	if err != nil {
		return nil, err
	}

	// build parameters
	if c.options.Namespace != "" && r.Params.Get("namespace") == "" {
		r.Params.Add("namespace", c.options.Namespace)
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
	req, err := http.NewRequestWithContext(c.requestContext(r), method, u.RequestURI(), body)
	if err != nil {
		return nil, err
	}

	// Optionally configure HTTP basic authentication
	if u.User != nil {
		username := u.User.Username()
		password, _ := u.User.Password()
		req.SetBasicAuth(username, password)
	} else if c.options.HTTPAuth != nil {
		req.SetBasicAuth(c.options.HTTPAuth.Username, c.options.HTTPAuth.Password)
	}

	// build headers
	req.Header = r.Header
	req.Header.Add("Accept-Encoding", "gzip")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.options.AppID != "" {
		req.Header.Set(apimodels.HTTPHeaderAppID, c.options.AppID)
		req.Header.Add("User-Agent", c.options.AppID)
	}
	for key, values := range c.options.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.URL.Host = u.Host
	req.URL.Scheme = u.Scheme
	req.Host = u.Host
	return req, nil
}

func (c *Client) requestContext(r *apimodels.HTTPRequest) context.Context {
	ctx := r.Ctx
	if ctx == nil {
		ctx = c.options.Context
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

// generate URL for a given endpoint
func (c *Client) url(endpoint string) (*url.URL, error) {
	base, _ := url.Parse(c.options.Address)
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
