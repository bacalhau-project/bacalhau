//go:build unit || !integration

package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// endpointHandler creates a standard HTTP handler for test endpoints
func endpointHandler(method string, statusCode int, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = fmt.Fprint(w, response)
	}
}

// defaultParams returns the default HTTP module parameters for testing
func defaultParams() Params {
	return Params{
		Network: &models.NetworkConfig{
			Type: models.NetworkHost,
		},
		Timeout:            5 * time.Second,
		MaxResponseSize:    1024 * 1024, // 1MB
		MemoryUsagePercent: 0.5,
	}
}

// testCase represents a standard HTTP test case configuration
type testCase struct {
	method        string
	path          string
	headers       string
	body          string
	networkType   models.Network
	hosts         []string
	expectSuccess bool
}
