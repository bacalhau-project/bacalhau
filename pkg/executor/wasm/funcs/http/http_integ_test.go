//go:build unit || !integration

package http

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/testdata"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// HTTPIntegrationSuite tests the HTTP module's integration with WASM.
// These tests verify the full functionality of the HTTP module when used
// from within a WASM environment, including network access, request handling,
// and error conditions.
type HTTPIntegrationSuite struct {
	suite.Suite
	ctx        context.Context
	runtime    wazero.Runtime
	testServer *httptest.Server
	serverURL  string
	wasmBytes  []byte
}

// SetupSuite sets up the test suite
func (s *HTTPIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()

	// Create a test HTTP server with standard endpoints
	mux := http.NewServeMux()
	endpoints := map[string]struct {
		method     string
		statusCode int
		response   string
	}{
		"/get":    {http.MethodGet, http.StatusOK, `{"message": "Hello, World!", "method": "GET"}`},
		"/post":   {http.MethodPost, http.StatusCreated, `{"message": "Resource created", "method": "POST"}`},
		"/put":    {http.MethodPut, http.StatusOK, `{"message": "Resource updated", "method": "PUT"}`},
		"/delete": {http.MethodDelete, http.StatusOK, `{"message": "Resource deleted", "method": "DELETE"}`},
	}

	for path, config := range endpoints {
		mux.HandleFunc(path, endpointHandler(config.method, config.statusCode, config.response))
	}

	// Echo endpoint with special handling
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"method": "%s", "received": true}`, r.Method)
	})

	s.testServer = httptest.NewServer(mux)
	s.serverURL = s.testServer.URL
	s.wasmBytes = testdata.Program()
}

// TearDownSuite tears down the test suite
func (s *HTTPIntegrationSuite) TearDownSuite() {
	if s.testServer != nil {
		s.testServer.Close()
	}
	if s.runtime != nil {
		s.runtime.Close(s.ctx)
	}
}

// SetupTest sets up each test
func (s *HTTPIntegrationSuite) SetupTest() {
	s.runtime = wazero.NewRuntimeWithConfig(s.ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
}

// TearDownTest tears down each test
func (s *HTTPIntegrationSuite) TearDownTest() {
	if s.runtime != nil {
		s.runtime.Close(s.ctx)
	}
}

// runWasmTest is a helper function to run a WASM test with the given environment variables
func (s *HTTPIntegrationSuite) runTestCase(tc testCase) {
	envVars := map[string]string{
		"HTTP_METHOD": tc.method,
		"HTTP_URL":    s.serverURL + tc.path,
	}

	params := defaultParams()
	params.Network = &models.NetworkConfig{
		Type:    tc.networkType,
		Domains: tc.hosts,
	}

	// Setup standard streams
	var stdout, stderr bytes.Buffer

	// Configure WASI
	wasiConfig := wazero.NewModuleConfig().
		WithArgs("test").
		WithSysWalltime().
		WithSysNanotime().
		WithRandSource(rand.New(rand.NewSource(time.Now().UnixNano()))).
		WithStdin(bytes.NewReader(nil)).
		WithStdout(&stdout).
		WithStderr(&stderr)

	// Add environment variables
	for key, value := range envVars {
		wasiConfig = wasiConfig.WithEnv(key, value)
	}

	// Create runtime and instantiate modules
	runtime := wazero.NewRuntimeWithConfig(s.ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	defer runtime.Close(s.ctx)

	// Instantiate WASI and HTTP module
	_, err := wasi_snapshot_preview1.Instantiate(s.ctx, runtime)
	require.NoError(s.T(), err, "Failed to instantiate WASI")

	if params.Network != nil && params.Network.Type != models.NetworkNone {
		err = InstantiateModule(s.ctx, runtime, params)
		require.NoError(s.T(), err, "Failed to instantiate HTTP module")
	}

	// Compile and instantiate the test module
	compiled, err := runtime.CompileModule(s.ctx, s.wasmBytes)
	require.NoError(s.T(), err, "Failed to compile WASM module")

	instance, err := runtime.InstantiateModule(s.ctx, compiled, wasiConfig)
	if err != nil {
		s.T().Logf("Failed to instantiate WASM module: %v", err)
		if tc.expectSuccess {
			require.NoError(s.T(), err, "Failed to instantiate WASM module")
		}
		return // Expected failure
	}
	defer instance.Close(s.ctx)

	// Run the module
	_, err = instance.ExportedFunction("_start").Call(s.ctx)

	// Log output
	s.T().Logf("Stdout: %s", stdout.String())
	s.T().Logf("Stderr: %s", stderr.String())

	// Verify results
	if tc.expectSuccess {
		if err != nil && !strings.Contains(err.Error(), "exit_code(0)") {
			require.NoError(s.T(), err, "Error calling _start function")
		}
	} else {
		if err == nil {
			assert.NotEmpty(s.T(), stderr.String(), "Expected error output but got none")
		} else if strings.Contains(err.Error(), "exit_code(0)") {
			assert.Fail(s.T(), "Expected failure but got success (exit code 0)")
		}
	}
}

// Test cases
func (s *HTTPIntegrationSuite) TestWasmHTTPMethods() {
	cases := []testCase{
		{networkType: models.NetworkHost, method: "GET", path: "/get", expectSuccess: true},
		{networkType: models.NetworkHost, method: "POST", path: "/post", headers: "Content-Type: application/json", body: `{"test": "data"}`, expectSuccess: true},
		{networkType: models.NetworkHost, method: "PUT", path: "/put", headers: "Content-Type: application/json", body: `{"test": "update"}`, expectSuccess: true},
		{networkType: models.NetworkHost, method: "DELETE", path: "/delete", expectSuccess: true},
	}

	for _, tc := range cases {
		s.Run(fmt.Sprintf("HTTP_%s", tc.method), func() {
			s.runTestCase(tc)
		})
	}
}

func (s *HTTPIntegrationSuite) TestWasmHTTPNetworkRestrictions() {
	cases := []testCase{
		{
			method: "GET", path: "/get",
			networkType:   models.NetworkNone,
			expectSuccess: false,
		},
		{
			method: "GET", path: "/get",
			networkType:   models.NetworkHTTP,
			hosts:         []string{"example.com", "test.com"},
			expectSuccess: false,
		},
	}

	for _, tc := range cases {
		s.Run(fmt.Sprintf("NetworkType_%s", tc.networkType), func() {
			s.runTestCase(tc)
		})
	}
}

func (s *HTTPIntegrationSuite) TestWasmHTTPWildcardHostMatching() {
	u, err := url.Parse(s.serverURL)
	require.NoError(s.T(), err, "Failed to parse server URL")
	host := u.Hostname()

	cases := []testCase{
		{
			method: "GET", path: "/get",
			networkType:   models.NetworkHTTP,
			hosts:         []string{host},
			expectSuccess: true,
		},
		{
			method: "GET", path: "/get",
			networkType:   models.NetworkHTTP,
			hosts:         []string{"127.0.0.*"},
			expectSuccess: true,
		},
	}

	for i, tc := range cases {
		s.Run(fmt.Sprintf("WildcardMatch_%d", i), func() {
			s.runTestCase(tc)
		})
	}
}

// Run the test suite
func TestHTTPIntegration(t *testing.T) {
	suite.Run(t, new(HTTPIntegrationSuite))
}
