//go:build unit || !integration

package semantic_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
)

func TestJobSelectionHttp(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		contentType string
		body        []byte
		expectBid   bool
		expectWait  bool
	}{
		{
			"fail the response and don't select the job",
			http.StatusInternalServerError,
			"text/plain",
			[]byte("500 - Something bad happened!"),
			false,
			false,
		},
		{
			"succeed the response and select the job",
			http.StatusOK,
			"text/plain",
			[]byte("200 - Everything is good!"),
			true,
			false,
		},
		{
			"pass a JSON response to select the job",
			http.StatusOK,
			"application/json",
			[]byte(`{"shouldBid": true, "reason": "looks like a lovely job"}`),
			true,
			false,
		},
		{
			"pass a JSON response to reject the job",
			http.StatusOK,
			"application/json",
			[]byte(`{"shouldBid": false, "reason": "this job really stinks!"}`),
			false,
			false,
		},
		{
			"pass a JSON response to wait for a future approval",
			http.StatusAccepted,
			"application/json",
			[]byte(`{"shouldWait": true, "reason": "gonna have to think about this one"}`),
			false,
			true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var requestPayload bidstrategy.JobSelectionPolicyProbeData

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, r.Method, "POST")
				// Try to decode the request body into the struct. If there is an error,
				// respond to the client with the error message and a 400 status code.
				err := json.NewDecoder(r.Body).Decode(&requestPayload)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				w.Header().Add("Content-Type", test.contentType)
				w.WriteHeader(test.status)
				w.Write(test.body)
			}))
			defer svr.Close()

			params := semantic.ExternalHTTPStrategyParams{URL: svr.URL}
			strategy := semantic.NewExternalHTTPStrategy(params)
			request := getBidStrategyRequest()
			result, err := strategy.ShouldBid(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, test.expectBid, result.ShouldBid)
			require.Equal(t, test.expectWait, result.ShouldWait)

			// this makes sure that the http payload was given to the http endpoint
			require.Equal(t, request.Job.Metadata.ID, requestPayload.JobID)
		})
	}
}
