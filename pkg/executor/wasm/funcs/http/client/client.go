// Package client provides a TinyGo-compatible client for the HTTP host functions.
package client

import (
	"sort"
	"strings"
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

// Default buffer sizes
const (
	DefaultHeadersBufferSize = 8 * 1024   // 8KB for headers
	DefaultBodyBufferSize    = 256 * 1024 // 256KB for body
)

// External function declarations for WASM imports
//
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

// Headers represents HTTP headers as a map
type Headers map[string][]string

// Response represents an HTTP response
type Response struct {
	StatusCode int     // HTTP status code
	Headers    Headers // Response headers
	Body       string  // Response body
}

// Add adds a header value for a given key
func (h Headers) Add(key, value string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if h[key] == nil {
		h[key] = []string{}
	}
	h[key] = append(h[key], value)
}

// Set sets a header value for a given key, replacing any existing values
func (h Headers) Set(key, value string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	h[key] = []string{value}
}

// Get returns the first value for a given header key
func (h Headers) Get(key string) string {
	if values := h[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// Values returns all values for a given header key
func (h Headers) Values(key string) []string {
	return h[key]
}

// String converts headers to a string format for internal use
func (h Headers) String() string {
	if len(h) == 0 {
		return ""
	}

	var headerLines []string
	for key, values := range h {
		for _, value := range values {
			headerLines = append(headerLines, key+": "+value)
		}
	}
	sort.Strings(headerLines) // ensure consistent ordering
	return strings.Join(headerLines, "\n")
}

// ParseHeaders parses a header string into a Headers map
func ParseHeaders(headerStr string) Headers {
	headers := make(Headers)
	if headerStr == "" {
		return headers
	}

	lines := strings.Split(headerStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headers.Add(key, value)
	}
	return headers
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
type Client struct {
	// HeadersBufferSize is the size of the buffer for response headers
	HeadersBufferSize uint32
	// BodyBufferSize is the size of the buffer for response body
	BodyBufferSize uint32
}

// NewClient creates a new HTTP client with default buffer sizes
func NewClient() *Client {
	return &Client{
		HeadersBufferSize: DefaultHeadersBufferSize,
		BodyBufferSize:    DefaultBodyBufferSize,
	}
}

// Get makes a GET request
func (c *Client) Get(url string, headers Headers) (*Response, error) {
	return c.Request(MethodGet, url, headers, "")
}

// Post makes a POST request
func (c *Client) Post(url string, headers Headers, body string) (*Response, error) {
	return c.Request(MethodPost, url, headers, body)
}

// Put makes a PUT request
func (c *Client) Put(url string, headers Headers, body string) (*Response, error) {
	return c.Request(MethodPut, url, headers, body)
}

// Delete makes a DELETE request
func (c *Client) Delete(url string, headers Headers) (*Response, error) {
	return c.Request(MethodDelete, url, headers, "")
}

// Head makes a HEAD request
func (c *Client) Head(url string, headers Headers) (*Response, error) {
	return c.Request(MethodHead, url, headers, "")
}

// Options makes an OPTIONS request
func (c *Client) Options(url string, headers Headers) (*Response, error) {
	return c.Request(MethodOptions, url, headers, "")
}

// Patch makes a PATCH request
func (c *Client) Patch(url string, headers Headers, body string) (*Response, error) {
	return c.Request(MethodPatch, url, headers, body)
}

// Request makes a HTTP request and returns the response
func (c *Client) Request(method uint32, url string, headers Headers, body string) (*Response, error) {
	urlPtr, urlLen := stringToPtr(url)
	headersPtr, headersLen := stringToPtr(headers.String())
	bodyPtr, bodyLen := stringToPtr(body)

	// Allocate buffers with the configured sizes
	responseHeaders := make([]byte, c.HeadersBufferSize)
	responseBody := make([]byte, c.BodyBufferSize)

	// Set up response variables
	var responseStatus uint32
	responseHeadersLen := c.HeadersBufferSize
	responseBodyLen := c.BodyBufferSize

	responseHeadersPtr := uint32(uintptr(unsafe.Pointer(&responseHeaders[0])))
	responseHeadersLenPtr := uint32(uintptr(unsafe.Pointer(&responseHeadersLen)))
	responseBodyPtr := uint32(uintptr(unsafe.Pointer(&responseBody[0])))
	responseBodyLenPtr := uint32(uintptr(unsafe.Pointer(&responseBodyLen)))
	responseStatusPtr := uint32(uintptr(unsafe.Pointer(&responseStatus)))

	// Make the request
	status := httpRequest(
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

	// Convert response to strings - use the actual lengths returned by the host
	respHeaders := string(responseHeaders[:responseHeadersLen])
	respBody := string(responseBody[:responseBodyLen])

	return &Response{
		StatusCode: int(responseStatus),
		Headers:    ParseHeaders(respHeaders),
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
