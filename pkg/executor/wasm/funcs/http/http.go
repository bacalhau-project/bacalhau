package http

import (
	"context"
	"errors"
	"io"
	"math"
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

// Error constants for HTTP module
const (
	errInvalidURL     = "invalid URL"
	errHostNotAllowed = "host not allowed"
	errInvalidBody    = "invalid body"
)

type Params struct {
	// Network defines the networking configuration including type and allowed domains
	Network *models.NetworkConfig

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
	if params.Network == nil || params.Network.Disabled() {
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

	// Ensure network config is normalized
	if params.Network != nil {
		params.Network.Normalize()
	}

	return &module{
		params: params,
		client: &http.Client{Timeout: params.Timeout},
	}
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
	// Verify output pointers are provided
	if responseHeadersPtr == 0 || responseHeadersLenPtr == 0 ||
		responseBodyPtr == 0 || responseBodyLenPtr == 0 {
		return StatusBadInput
	}

	// Prepare the request
	req, err := m.prepareHTTPRequest(ctx, mod, method, urlPtr, urlLen, headersPtr, headersLen, bodyPtr, bodyLen)
	if err != nil {
		switch err.Error() {
		case errInvalidURL:
			return StatusInvalidURL
		case errHostNotAllowed:
			return StatusNotAllowed
		case errInvalidBody:
			return StatusBadInput
		default:
			return StatusInvalidURL
		}
	}

	// Execute request
	resp, err := m.client.Do(req)
	if err != nil {
		// Check for timeout
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return StatusTimeout
		}
		return StatusNetworkError
	}
	defer func() { _ = resp.Body.Close() }()

	// Calculate maximum allowed response size and read response with limit
	maxSize := m.calculateMaxResponseSize(mod)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, safeInt64(maxSize)))
	if err != nil {
		return StatusNetworkError
	}

	// Write response to memory
	return m.writeResponseToMemory(mod, resp, respBody,
		responseHeadersPtr, responseHeadersLenPtr,
		responseBodyPtr, responseBodyLenPtr,
		statusPtr)
}

// isHostAllowed checks if the given host is allowed according to the configuration
func (m *module) isHostAllowed(host string) bool {
	if m.params.Network == nil {
		return false
	}

	if m.params.Network.Type == models.NetworkFull || m.params.Network.Type == models.NetworkHost {
		return true
	}

	if m.params.Network.Type == models.NetworkHTTP {
		allowedDomains := m.params.Network.DomainSet()
		for _, allowed := range allowedDomains {
			if matched, _ := matchWildcard(allowed, host); matched {
				return true
			}
		}
	}

	return false
}

// matchWildcard checks if a hostname matches a pattern with wildcards
// Supports simple glob patterns like "*.example.com" or "api.*.org"
// Ports are stripped from both pattern and host before matching
func matchWildcard(pattern, host string) (bool, error) {
	// Strip port number from both pattern and host if present
	if colonIndex := strings.LastIndex(pattern, ":"); colonIndex != -1 {
		pattern = pattern[:colonIndex]
	}
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

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

// prepareHTTPRequest prepares the HTTP request from WASM memory
func (m *module) prepareHTTPRequest(
	ctx context.Context,
	mod api.Module,
	method uint32,
	urlPtr, urlLen uint32,
	headersPtr, headersLen uint32,
	bodyPtr, bodyLen uint32,
) (*http.Request, error) {
	memory := mod.Memory()

	// Read URL from WebAssembly memory
	urlBytes, ok := memory.Read(urlPtr, urlLen)
	if !ok {
		return nil, errors.New(errInvalidURL)
	}

	urlStr := string(urlBytes)
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.New(errInvalidURL)
	}

	// Check if the host is allowed
	if !m.isHostAllowed(parsedURL.Host) {
		return nil, errors.New(errHostNotAllowed)
	}

	// Read headers if provided
	headers := make(http.Header)
	if headersPtr != 0 && headersLen > 0 {
		var headersBytes []byte
		if headersBytes, ok = memory.Read(headersPtr, headersLen); ok {
			parseHeaders(headersBytes, headers)
		}
	}

	// Read body if provided
	var bodyBytes []byte
	if bodyPtr != 0 && bodyLen > 0 {
		bodyBytes, ok = memory.Read(bodyPtr, bodyLen)
		if !ok {
			return nil, errors.New(errInvalidBody)
		}
	}

	methodStr := methodToString(method)

	var reqBody io.Reader
	if len(bodyBytes) > 0 {
		reqBody = strings.NewReader(string(bodyBytes))
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

// writeResponseToMemory writes the HTTP response back to WASM memory
func (m *module) writeResponseToMemory(
	mod api.Module,
	resp *http.Response,
	respBody []byte,
	responseHeadersPtr, responseHeadersLenPtr uint32,
	responseBodyPtr, responseBodyLenPtr uint32,
	statusPtr uint32,
) uint32 {
	memory := mod.Memory()

	// Format response headers as a string
	var headerLines []string
	for k, v := range resp.Header {
		for _, value := range v {
			headerLines = append(headerLines, k+": "+value)
		}
	}
	headersStr := strings.Join(headerLines, "\n")

	// Read the maximum available buffer sizes
	headersBufSize, ok := memory.ReadUint32Le(responseHeadersLenPtr)
	if !ok {
		return StatusMemoryError
	}

	bodyBufSize, ok := memory.ReadUint32Le(responseBodyLenPtr)
	if !ok {
		return StatusMemoryError
	}

	// Truncate data if necessary to fit in provided buffers
	if safeUint32(len(headersStr)) > headersBufSize {
		headersStr = headersStr[:headersBufSize]
	}

	if safeUint32(len(respBody)) > bodyBufSize {
		respBody = respBody[:bodyBufSize]
	}

	// Write response status code if pointer provided
	if statusPtr != 0 {
		if !memory.WriteUint32Le(statusPtr, safeUint32(resp.StatusCode)) {
			return StatusMemoryError
		}
	}

	// Write actual lengths
	if !memory.WriteUint32Le(responseHeadersLenPtr, safeUint32(len(headersStr))) {
		return StatusMemoryError
	}

	if !memory.WriteUint32Le(responseBodyLenPtr, safeUint32(len(respBody))) {
		return StatusMemoryError
	}

	// Write headers and body data
	if !memory.Write(responseHeadersPtr, []byte(headersStr)) {
		return StatusMemoryError
	}

	if !memory.Write(responseBodyPtr, respBody) {
		return StatusMemoryError
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

// safeInt64 converts uint64 to int64, clamping to MaxInt64 if necessary
func safeInt64(n uint64) int64 {
	if n > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(n)
}

// safeUint32 converts int to uint32, clamping to 0 or MaxUint32 if necessary
func safeUint32(n int) uint32 {
	if n < 0 {
		return 0
	}
	// Use uint64 for comparison to handle all architectures safely
	if uint64(n) > uint64(^uint32(0)) {
		return ^uint32(0) // MaxUint32 using bitwise complement of zero
	}
	// Safe to convert now that we've clamped the value
	return uint32(n) //nolint:gosec // Safe because we've clamped the value to valid uint32 range
}
