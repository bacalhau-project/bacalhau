// Package client provides a TinyGo-compatible client for the HTTP host functions.
package client

import (
	"unsafe"
)

// HTTP request method constants
const (
	MethodGet     uint32 = 0
	MethodPost    uint32 = 1
	MethodPut     uint32 = 2
	MethodDelete  uint32 = 3
	MethodHead    uint32 = 4
	MethodOptions uint32 = 5
	MethodPatch   uint32 = 6
)

// Result codes
const (
	StatusSuccess      uint32 = 0
	StatusInvalidURL   uint32 = 1
	StatusNetworkError uint32 = 2
	StatusTimeout      uint32 = 3
	StatusNotAllowed   uint32 = 4
	StatusTooLarge     uint32 = 5
	StatusBadInput     uint32 = 6
	StatusMemoryError  uint32 = 7
)

// External function declarations for WASM imports
//
//go:wasmimport wasi:http/requests http_get_response_size
func httpGetResponseSize(method uint32, urlPtr, urlLen uint32, headersPtr, headersLen uint32) (status uint32, size uint32)

//go:wasmimport wasi:http/requests http_request
func httpRequest(
	method uint32,
	urlPtr, urlLen uint32,
	headersPtr, headersLen uint32,
	bodyPtr, bodyLen uint32,
	responseHeadersPtr, responseHeadersLenPtr uint32,
	responseBodyPtr, responseBodyLenPtr uint32,
	statusPtr uint32,
) uint32

// Response represents an HTTP response
type Response struct {
	StatusCode int    // HTTP status code
	Headers    string // Response headers
	Body       string // Response body
}

// HTTPError represents an error from the HTTP client
type HTTPError struct {
	Code    uint32
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// Client is a HTTP client for making requests using the host functions
type Client struct{}

// NewClient creates a new HTTP client
func NewClient() *Client {
	return &Client{}
}

// GetResponseSize gets the size of the response for a HTTP request
func (c *Client) GetResponseSize(method uint32, url string, headers string) (uint32, uint32) {
	urlPtr, urlLen := stringToPtr(url)
	headersPtr, headersLen := stringToPtr(headers)

	return httpGetResponseSize(method, urlPtr, urlLen, headersPtr, headersLen)
}

// Request makes a HTTP request and returns the response
func (c *Client) Request(method uint32, url, headers, body string) (*Response, error) {
	urlPtr, urlLen := stringToPtr(url)
	headersPtr, headersLen := stringToPtr(headers)
	bodyPtr, bodyLen := stringToPtr(body)

	// Allocate memory for response headers and body
	var responseStatus uint32
	var responseHeadersLen, responseBodyLen uint32

	// First get the response size
	status, size := httpGetResponseSize(method, urlPtr, urlLen, headersPtr, headersLen)
	if status != StatusSuccess {
		return nil, errorFromStatus(status)
	}

	// Allocate memory for response
	responseHeaders := make([]byte, 1024) // Reasonable default for headers
	responseBody := make([]byte, size)

	responseHeadersPtr := uint32(uintptr(unsafe.Pointer(&responseHeaders[0])))
	responseHeadersLenPtr := uint32(uintptr(unsafe.Pointer(&responseHeadersLen)))
	responseBodyPtr := uint32(uintptr(unsafe.Pointer(&responseBody[0])))
	responseBodyLenPtr := uint32(uintptr(unsafe.Pointer(&responseBodyLen)))
	responseStatusPtr := uint32(uintptr(unsafe.Pointer(&responseStatus)))

	// Make the request
	status = httpRequest(
		method,
		urlPtr, urlLen,
		headersPtr, headersLen,
		bodyPtr, bodyLen,
		responseHeadersPtr, responseHeadersLenPtr,
		responseBodyPtr, responseBodyLenPtr,
		responseStatusPtr,
	)

	if status != StatusSuccess {
		return nil, errorFromStatus(status)
	}

	// Convert response to strings
	respHeaders := string(responseHeaders[:responseHeadersLen])
	respBody := string(responseBody[:responseBodyLen])

	return &Response{
		StatusCode: int(responseStatus),
		Headers:    respHeaders,
		Body:       respBody,
	}, nil
}

// Helper functions

// stringToPtr converts a string to a pointer and length
func stringToPtr(s string) (uint32, uint32) {
	if s == "" {
		return 0, 0
	}
	bytes := []byte(s)
	return uint32(uintptr(unsafe.Pointer(&bytes[0]))), uint32(len(bytes))
}

// errorFromStatus converts a status code to an error
func errorFromStatus(status uint32) error {
	switch status {
	case StatusSuccess:
		return nil
	case StatusInvalidURL:
		return &HTTPError{Code: status, Message: "invalid URL"}
	case StatusNetworkError:
		return &HTTPError{Code: status, Message: "network error"}
	case StatusTimeout:
		return &HTTPError{Code: status, Message: "request timeout"}
	case StatusNotAllowed:
		return &HTTPError{Code: status, Message: "host not allowed"}
	case StatusTooLarge:
		return &HTTPError{Code: status, Message: "response too large"}
	case StatusBadInput:
		return &HTTPError{Code: status, Message: "bad input"}
	case StatusMemoryError:
		return &HTTPError{Code: status, Message: "memory error"}
	default:
		return &HTTPError{Code: status, Message: "unknown error"}
	}
}

// Convenience methods for common HTTP methods

// Get makes a GET request
func (c *Client) Get(url, headers string) (*Response, error) {
	return c.Request(MethodGet, url, headers, "")
}

// Post makes a POST request
func (c *Client) Post(url, headers, body string) (*Response, error) {
	return c.Request(MethodPost, url, headers, body)
}

// Put makes a PUT request
func (c *Client) Put(url, headers, body string) (*Response, error) {
	return c.Request(MethodPut, url, headers, body)
}

// Delete makes a DELETE request
func (c *Client) Delete(url, headers string) (*Response, error) {
	return c.Request(MethodDelete, url, headers, "")
}
