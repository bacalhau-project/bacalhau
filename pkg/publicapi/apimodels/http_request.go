package apimodels

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// HTTPRequest is used to help build up a request
type HTTPRequest struct {
	Params  url.Values
	Body    io.Reader
	BodyObj interface{}
	Ctx     context.Context
	Header  http.Header

	// A good place to define other fields that are common to all http requests,
	// such as auth tokens
}

// NewHTTPRequest is used to create a new request
func NewHTTPRequest() *HTTPRequest {
	r := &HTTPRequest{
		Header: make(http.Header),
		Params: make(map[string][]string),
	}
	return r
}

// HTTPBasicAuth is used to authenticate http client with HTTP Basic Authentication
type HTTPBasicAuth struct {
	// Username to use for HTTP Basic Authentication
	Username string

	// Password to use for HTTP Basic Authentication
	Password string
}
