//go:build !integration

package computenode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"

	"github.com/filecoin-project/bacalhau/pkg/computenode/tooling"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

func getProbeDataWithVolume() JobSelectionPolicyProbeData {
	return JobSelectionPolicyProbeData{
		NodeID: "node-id",
		JobID:  "job-id",
		Spec: model.Spec{
			Inputs: []model.StorageSpec{
				{
					StorageSource: model.StorageSourceIPFS,
					CID:           "volume-id",
				},
			},
		},
	}
}

func TestJobSelectionPolicy(t *testing.T) {

	testCases := []struct {
		name              string
		expectedResult    bool
		hasStorageLocally bool
		policy            JobSelectionPolicy
		data              JobSelectionPolicyProbeData
	}{
		// we don't allow stateless jobs and we try to run a stateless job
		{
			"stateless job -> disallowed correct",
			false,
			false,
			JobSelectionPolicy{
				RejectStatelessJobs: true,
			},
			JobSelectionPolicyProbeData{},
		},

		// we don't allow stateless jobs and we try to run a stateless job
		{
			"stateless job -> allowed correct",
			true,
			false,
			JobSelectionPolicy{
				RejectStatelessJobs: false,
			},
			JobSelectionPolicyProbeData{},
		},

		// we are local - we do have the file - we should accept
		{
			"local mode -> have file -> should accept",
			true,
			true,
			JobSelectionPolicy{
				RejectStatelessJobs: true,
				Locality:            Local,
			},
			getProbeDataWithVolume(),
		},

		// we are local - we don't have the file - we should reject
		{
			"local mode -> don't have file -> should reject",
			false,
			false,
			JobSelectionPolicy{
				RejectStatelessJobs: true,
				Locality:            Local,
			},
			getProbeDataWithVolume(),
		},

		// we are anywhere - we do have the file - we should accept
		{
			"anywhere mode -> have file -> should accept",
			true,
			true,
			JobSelectionPolicy{
				RejectStatelessJobs: true,
				Locality:            Anywhere,
			},
			getProbeDataWithVolume(),
		},

		// we are anywhere - we don't have the file - we should accept
		{
			"anywhere mode -> don't have file -> should accept",
			true,
			false,
			JobSelectionPolicy{
				RejectStatelessJobs: true,
				Locality:            Anywhere,
			},
			getProbeDataWithVolume(),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			executor, err := noop_executor.NewNoopExecutorWithConfig(tooling.HasStorageNoopExecutorConfig(test.hasStorageLocally))
			require.NoError(t, err)
			result, err := ApplyJobSelectionPolicy(
				context.Background(),
				test.policy,
				executor,
				test.data,
			)
			require.NoError(t, err)
			require.Equal(t, test.expectedResult, result)
		})
	}

}

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
			executor, err := noop_executor.NewNoopExecutorWithConfig(tooling.BlankNoopExecutorConfig())

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
			require.NoError(t, err)
			result, err := ApplyJobSelectionPolicy(
				context.Background(),
				JobSelectionPolicy{
					ProbeHTTP: svr.URL,
				},
				executor,
				getProbeDataWithVolume(),
			)
			require.NoError(t, err)
			require.Equal(t, test.expectedResult, result)

			// this makes sure that the http payload was given to the http endpoint
			require.Equal(t, "job-id", requestPayload.JobID)
		})
	}
}

func TestJobSelectionExec(t *testing.T) {
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
			executor, err := noop_executor.NewNoopExecutorWithConfig(tooling.BlankNoopExecutorConfig())
			require.NoError(t, err)
			command := "exit 0"
			if test.failMode {
				command = "exit 1"
			}
			result, err := ApplyJobSelectionPolicy(
				context.Background(),
				JobSelectionPolicy{
					ProbeExec: command,
				},
				executor,
				getProbeDataWithVolume(),
			)
			require.NoError(t, err)
			require.Equal(t, test.expectedResult, result)
		})
	}
}
