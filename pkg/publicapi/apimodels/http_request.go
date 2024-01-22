package apimodels

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

type HTTPCredential struct {
	// An HTTP authorization scheme, such as one registered with IANA
	// https://www.iana.org/assignments/http-authschemes/http-authschemes.xhtml
	Scheme string

	// For authorization schemes that only provide a single value, such as
	// Basic, the single string value providing the credential
	Value string

	// For authorization schemes that provide multiple values, a map of names to
	// values providing the credential
	Params map[string]string
}

func (cred HTTPCredential) String() string {
	strs := []string{cred.Scheme}
	if cred.Value != "" {
		strs = append(strs, cred.Value)
	}
	for key, value := range cred.Params {
		strs = append(strs, fmt.Sprintf("%s=%q", key, value))
	}
	return strings.Join(strs, " ")
}
