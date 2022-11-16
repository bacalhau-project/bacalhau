//go:build unit || !integration

package bidstrategy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobSelectionHttp(t *testing.T) {
	testCases := []struct {
		name           string
		failMode       bool
		expectedResult bool
	}{
		{
			"fail the response and don't select the job",
			true,
			false,
		},
		{
			"succeed the response and select the job",
			false,
			true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var requestPayload JobSelectionPolicyProbeData

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, r.Method, "POST")
				// Try to decode the request body into the struct. If there is an error,
				// respond to the client with the error message and a 400 status code.
				err := json.NewDecoder(r.Body).Decode(&requestPayload)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				if test.failMode {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("500 - Something bad happened!"))
				} else {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("200 - Everything is good!"))
				}
			}))
			defer svr.Close()

			params := ExternalHTTPStrategyParams{URL: svr.URL}
			strategy := NewExternalHTTPStrategy(params)
			request := getBidStrategyRequest()
			result, err := strategy.ShouldBid(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, test.expectedResult, result.ShouldBid)

			// this makes sure that the http payload was given to the http endpoint
			require.Equal(t, request.Job.ID, requestPayload.JobID)
		})
	}
}
