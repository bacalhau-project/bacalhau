package http

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ModuleName Defines standard HTTP function namespace
const ModuleName = "wasi:http/requests"

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

const (
	// DefaultTimeout is the default timeout for HTTP requests
	DefaultTimeout = 30 * time.Second

	// DefaultMaxResponseSize is the default maximum response size
	DefaultMaxResponseSize = 100 * 1024 * 1024 // 100MB

	// MinResponseSize is the minimum response size allowed, in case the job didn't specify memory allocation
	MinResponseSize = 64 * 1024 // 64Kb

	// DefaultMemoryUsagePercent is the default percentage of memory that can be used for HTTP responses
	DefaultMemoryUsagePercent = 0.8 // 80%
)

type Params struct {
	// NetworkType defines the networking mode: "none", "http", "host", "full"
	NetworkType models.Network

	// AllowedHosts is a list of allowed hostnames when NetworkType is "http"
	AllowedHosts []string

	// Timeout specifies the maximum duration for HTTP requests
	Timeout time.Duration

	// MaxResponseSize is the maximum allowed response size
	// Default is 100MB
	MaxResponseSize uint64

	// MemoryUsagePercent is the percentage of available memory that can be used for HTTP responses
	// Default is 80%
	MemoryUsagePercent float64
}

// InstantiateModule instantiates the HTTP host functions
func InstantiateModule(ctx context.Context, r wazero.Runtime, params Params) error {
	if params.NetworkType == models.NetworkNone {
		return nil // Don't register any network functions
	}

	httpModule := newHTTPModule(params)

	// Create module builder
	moduleBuilder := r.NewHostModuleBuilder(ModuleName)

	// Register http_request function
	moduleBuilder.NewFunctionBuilder().
		WithFunc(httpModule.httpRequest).
		WithName("http_request").
		WithParameterNames("method", "url_ptr", "url_len", "headers_ptr", "headers_len",
			"body_ptr", "body_len", "response_headers_ptr", "response_headers_len_ptr",
			"response_body_ptr", "response_body_len_ptr",
			"status_ptr", // This is for the HTTP status code
		).
		WithResultNames("status_code"). // This is the function's success/error code
		Export("http_request")

	// Instantiate the module
	_, err := moduleBuilder.Instantiate(ctx)
	return err
}

// module manages HTTP functionality for WASM
type module struct {
	params Params
	client *http.Client
}

// newHTTPModule creates a new HTTP module instance
func newHTTPModule(params Params) *module {
	if params.Timeout == 0 {
		params.Timeout = DefaultTimeout
	}
	if params.MaxResponseSize == 0 {
		params.MaxResponseSize = DefaultMaxResponseSize
	}
	if params.MemoryUsagePercent == 0 {
		params.MemoryUsagePercent = DefaultMemoryUsagePercent
	}
	if params.AllowedHosts == nil {
		params.AllowedHosts = []string{}
	}
	return &module{
		params: params,
		client: &http.Client{
			Timeout: params.Timeout,
		},
	}
}

// isHostAllowed checks if the given host is allowed according to the configuration
func (m *module) isHostAllowed(host string) bool {
	if m.params.NetworkType == models.NetworkFull || m.params.NetworkType == models.NetworkHost {
		return true
	}

	if m.params.NetworkType == models.NetworkHTTP {
		for _, allowed := range m.params.AllowedHosts {
			if matched, _ := matchWildcard(allowed, host); matched {
				return true
			}
		}
	}

	return false
}

// calculateMaxResponseSize determines the maximum allowable response size
// based on the module's memory and configuration
func (m *module) calculateMaxResponseSize(mod api.Module) uint64 {
	// Get the module's total memory size
	memorySize := mod.Memory().Size()

	// Calculate allowable size based on configured percentage
	maxAllowableSize := uint64(float64(memorySize) * m.params.MemoryUsagePercent)

	// Enforce absolute maximum to prevent extremely large allocations
	if maxAllowableSize > m.params.MaxResponseSize {
		maxAllowableSize = m.params.MaxResponseSize
	}

	// Enforce minimum size
	if maxAllowableSize < MinResponseSize {
		maxAllowableSize = MinResponseSize
	}

	return maxAllowableSize
}

// matchWildcard checks if a hostname matches a pattern with wildcards
// Supports simple glob patterns like "*.example.com" or "api.*.org"
func matchWildcard(pattern, host string) (bool, error) {
	if pattern == "*" {
		return true, nil
	}

	if !strings.Contains(pattern, "*") {
		return pattern == host, nil
	}

	parts := strings.Split(pattern, "*")
	hostLower := strings.ToLower(host)

	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(hostLower, parts[1]), nil
	}

	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(hostLower, parts[0]), nil
	}

	return strings.HasPrefix(hostLower, parts[0]) && strings.HasSuffix(hostLower, parts[1]), nil
}

// prepareRequest creates and prepares an HTTP request
func (m *module) prepareRequest(
	ctx context.Context,
	method uint32,
	urlStr string,
	headers http.Header,
	body []byte,
) (*http.Request, error) {
	methodStr := methodToString(method)

	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, methodStr, urlStr, reqBody)
	if err != nil {
		return nil, err
	}

	// Set headers
	for k, v := range headers {
		for _, hv := range v {
			req.Header.Add(k, hv)
		}
	}

	return req, nil
}

// httpRequest implements the WASI HTTP request function
func (m *module) httpRequest(
	ctx context.Context,
	mod api.Module,
	method uint32,
	urlPtr, urlLen uint32,
	headersPtr, headersLen uint32,
	bodyPtr, bodyLen uint32,
	responseHeadersPtr, responseHeadersLenPtr uint32,
	responseBodyPtr, responseBodyLenPtr uint32,
	statusPtr uint32,
) uint32 {
	memory := mod.Memory()

	// Read URL from WebAssembly memory
	urlBytes, ok := memory.Read(urlPtr, urlLen)
	if !ok {
		return StatusInvalidURL
	}

	urlStr := string(urlBytes)
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return StatusInvalidURL
	}

	// Check if the host is allowed
	if !m.isHostAllowed(parsedURL.Host) {
		return StatusNotAllowed
	}

	// Verify output pointers are provided
	if responseHeadersPtr == 0 || responseHeadersLenPtr == 0 ||
		responseBodyPtr == 0 || responseBodyLenPtr == 0 {
		return StatusBadInput
	}

	// Read headers if provided
	headers := make(http.Header)
	if headersPtr != 0 && headersLen > 0 {
		headersBytes, ok := memory.Read(headersPtr, headersLen)
		if ok {
			parseHeaders(headersBytes, headers)
		}
	}

	// Read body if provided
	var bodyBytes []byte
	if bodyPtr != 0 && bodyLen > 0 {
		bodyBytes, ok = memory.Read(bodyPtr, bodyLen)
		if !ok {
			return StatusBadInput
		}
	}

	// Calculate maximum allowed response size for this module
	maxSize := m.calculateMaxResponseSize(mod)

	// Prepare and execute the actual request
	req, err := m.prepareRequest(ctx, method, urlStr, headers, bodyBytes)
	if err != nil {
		return StatusInvalidURL
	}

	// Execute request
	resp, err := m.client.Do(req)
	if err != nil {
		// Check for timeout
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return StatusTimeout
		}
		return StatusNetworkError
	}
	defer resp.Body.Close()

	// Read response body with size limit
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSize)))
	if err != nil {
		return StatusNetworkError
	}

	// Format response headers as a string
	var headerLines []string
	for k, v := range resp.Header {
		for _, value := range v {
			headerLines = append(headerLines, k+": "+value)
		}
	}
	headersStr := strings.Join(headerLines, "\n")

	// Read the maximum available buffer sizes
	var headersBufSize, bodyBufSize uint32
	headersBufSize, ok = memory.ReadUint32Le(responseHeadersLenPtr)
	if !ok {
		return StatusMemoryError
	}

	bodyBufSize, ok = memory.ReadUint32Le(responseBodyLenPtr)
	if !ok {
		return StatusMemoryError
	}

	// Get actual sizes
	actualHeadersLen := uint32(len(headersStr))
	actualBodyLen := uint32(len(respBody))

	// Truncate if necessary and update the actual lengths
	if actualHeadersLen > headersBufSize {
		headersStr = headersStr[:headersBufSize]
		actualHeadersLen = headersBufSize
	}

	if actualBodyLen > bodyBufSize {
		respBody = respBody[:bodyBufSize]
		actualBodyLen = bodyBufSize
	}

	// Write response status code if pointer provided
	if statusPtr != 0 {
		ok = memory.WriteUint32Le(statusPtr, uint32(resp.StatusCode))
		if !ok {
			return StatusMemoryError
		}
	}

	// Write actual response header length
	ok = memory.WriteUint32Le(responseHeadersLenPtr, actualHeadersLen)
	if !ok {
		return StatusMemoryError
	}

	// Write actual response body length
	ok = memory.WriteUint32Le(responseBodyLenPtr, actualBodyLen)
	if !ok {
		return StatusMemoryError
	}

	// Write headers to memory
	for i, b := range []byte(headersStr) {
		ok = memory.WriteByte(responseHeadersPtr+uint32(i), b)
		if !ok {
			return StatusMemoryError
		}
	}

	// Write response body to memory
	for i, b := range respBody {
		ok = memory.WriteByte(responseBodyPtr+uint32(i), b)
		if !ok {
			return StatusMemoryError
		}
	}

	return StatusSuccess
}

// Helper functions

// methodToString converts method constant to string
func methodToString(method uint32) string {
	switch method {
	case MethodGet:
		return http.MethodGet
	case MethodPost:
		return http.MethodPost
	case MethodPut:
		return http.MethodPut
	case MethodDelete:
		return http.MethodDelete
	case MethodHead:
		return http.MethodHead
	case MethodOptions:
		return http.MethodOptions
	case MethodPatch:
		return http.MethodPatch
	default:
		return http.MethodGet
	}
}

// parseHeaders parses header bytes into HTTP headers
func parseHeaders(headerBytes []byte, headers http.Header) {
	headerStr := string(headerBytes)
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
}
