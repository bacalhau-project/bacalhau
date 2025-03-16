package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// HTTPUnitSuite contains unit tests for the HTTP module's core functionality.
// Tests here focus on individual components like network access control,
// wildcard matching, header parsing, and request preparation.
type HTTPUnitSuite struct {
	suite.Suite
	ctx        context.Context
	testServer *httptest.Server
	serverURL  string
	module     *module
}

// SetupSuite sets up the test suite
func (s *HTTPUnitSuite) SetupSuite() {
	s.ctx = context.Background()

	// Create a test HTTP server with standard endpoints
	mux := http.NewServeMux()
	endpoints := map[string]struct {
		method     string
		statusCode int
		response   string
	}{
		"/get":    {http.MethodGet, http.StatusOK, `{"message": "Hello, World!"}`},
		"/post":   {http.MethodPost, http.StatusCreated, `{"message": "Created"}`},
		"/put":    {http.MethodPut, http.StatusOK, `{"message": "Updated"}`},
		"/delete": {http.MethodDelete, http.StatusOK, `{"message": "Deleted"}`},
	}

	for path, config := range endpoints {
		mux.HandleFunc(path, endpointHandler(config.method, config.statusCode, config.response))
	}

	s.testServer = httptest.NewServer(mux)
	s.serverURL = s.testServer.URL
}

// TearDownSuite tears down the test suite
func (s *HTTPUnitSuite) TearDownSuite() {
	if s.testServer != nil {
		s.testServer.Close()
	}
}

// SetupTest sets up each test
func (s *HTTPUnitSuite) SetupTest() {
	s.module = newHTTPModule(defaultParams())
}

// TestNetworkAccess tests network access control based on network type and host allowlist
func (s *HTTPUnitSuite) TestNetworkAccess() {
	cases := []struct {
		name        string
		networkType models.Network
		hosts       []string
		testHost    string
		allowed     bool
	}{
		{
			name:        "Full network access allows any host",
			networkType: models.NetworkFull,
			testHost:    "example.com",
			allowed:     true,
		},
		{
			name:        "Host network access allows any host",
			networkType: models.NetworkHost,
			testHost:    "example.com",
			allowed:     true,
		},
		{
			name:        "No network access blocks all hosts",
			networkType: models.NetworkNone,
			testHost:    "example.com",
			allowed:     false,
		},
		{
			name:        "HTTP restricted - exact host match",
			networkType: models.NetworkHTTP,
			hosts:       []string{"example.com"},
			testHost:    "example.com",
			allowed:     true,
		},
		{
			name:        "HTTP restricted - host not in allowlist",
			networkType: models.NetworkHTTP,
			hosts:       []string{"example.com"},
			testHost:    "test.com",
			allowed:     false,
		},
		{
			name:        "HTTP restricted - wildcard match",
			networkType: models.NetworkHTTP,
			hosts:       []string{"*.example.com"},
			testHost:    "api.example.com",
			allowed:     true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			module := newHTTPModule(Params{
				Network: &models.NetworkConfig{
					Type:    tc.networkType,
					Domains: tc.hosts,
				},
			})
			assert.Equal(s.T(), tc.allowed, module.isHostAllowed(tc.testHost))
		})
	}
}

// TestWildcardMatching tests hostname wildcard pattern matching
func (s *HTTPUnitSuite) TestWildcardMatching() {
	cases := []struct {
		name    string
		pattern string
		host    string
		matches bool
	}{
		{
			name:    "Exact match",
			pattern: "example.com",
			host:    "example.com",
			matches: true,
		},
		{
			name:    "Exact match with port",
			pattern: "example.com",
			host:    "example.com:8080",
			matches: true,
		},
		{
			name:    "Prefix wildcard",
			pattern: "*.example.com",
			host:    "api.example.com",
			matches: true,
		},
		{
			name:    "Suffix wildcard",
			pattern: "api.*",
			host:    "api.example.com",
			matches: true,
		},
		{
			name:    "Middle wildcard",
			pattern: "api.*.com",
			host:    "api.test.com",
			matches: true,
		},
		{
			name:    "No match",
			pattern: "example.com",
			host:    "test.com",
			matches: false,
		},
		{
			name:    "Global wildcard",
			pattern: "*",
			host:    "anything.com",
			matches: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			matches, err := matchWildcard(tc.pattern, tc.host)
			assert.NoError(s.T(), err)
			assert.Equal(s.T(), tc.matches, matches)
		})
	}
}

// TestHTTPMethodConversion tests HTTP method constant conversion
func (s *HTTPUnitSuite) TestHTTPMethodConversion() {
	cases := []struct {
		method uint32
		str    string
	}{
		{MethodGet, http.MethodGet},
		{MethodPost, http.MethodPost},
		{MethodPut, http.MethodPut},
		{MethodDelete, http.MethodDelete},
		{MethodHead, http.MethodHead},
		{MethodOptions, http.MethodOptions},
		{MethodPatch, http.MethodPatch},
		{99, http.MethodGet}, // Invalid method defaults to GET
	}

	for _, tc := range cases {
		s.Run(tc.str, func() {
			assert.Equal(s.T(), tc.str, methodToString(tc.method))
		})
	}
}

// TestHeaderParsing tests HTTP header parsing
func (s *HTTPUnitSuite) TestHeaderParsing() {
	cases := []struct {
		name     string
		input    string
		expected http.Header
	}{
		{
			name:  "Single header",
			input: "Content-Type: application/json",
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name:  "Multiple headers",
			input: "Content-Type: application/json\nAuthorization: Bearer token",
			expected: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer token"},
			},
		},
		{
			name:  "Empty lines",
			input: "\nContent-Type: application/json\n\nAuthorization: Bearer token\n",
			expected: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer token"},
			},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: http.Header{},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			headers := make(http.Header)
			parseHeaders([]byte(tc.input), headers)
			assert.Equal(s.T(), tc.expected, headers)
		})
	}
}

// TestRequestPreparation tests HTTP request preparation
func (s *HTTPUnitSuite) TestRequestPreparation() {
	cases := []struct {
		name    string
		method  uint32
		url     string
		headers http.Header
		body    []byte
	}{
		{
			name:   "GET request",
			method: MethodGet,
			url:    s.serverURL + "/get",
		},
		{
			name:   "POST request with body",
			method: MethodPost,
			url:    s.serverURL + "/post",
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			body: []byte(`{"test":"data"}`),
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			req, err := s.module.prepareRequest(s.ctx, tc.method, tc.url, tc.headers, tc.body)
			require.NoError(s.T(), err)
			assert.Equal(s.T(), methodToString(tc.method), req.Method)
			assert.Equal(s.T(), tc.url, req.URL.String())
			if tc.headers != nil {
				for k, v := range tc.headers {
					assert.Equal(s.T(), v[0], req.Header.Get(k))
				}
			}
		})
	}
}

// Run the test suite
func TestHTTPUnit(t *testing.T) {
	suite.Run(t, new(HTTPUnitSuite))
}
