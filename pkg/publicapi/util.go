package publicapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
)

// SetupTests sets up a client for a requester node's API server, for testing.
func SetupTests(t *testing.T) *APIClient {
	ipt, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)

	rn, err := requestor_node.NewRequesterNode(ipt)
	assert.NoError(t, err)

	port, err := freeport.GetFreePort()
	assert.NoError(t, err)

	s := NewServer(rn, "0.0.0.0", port)
	c := NewAPIClient(s.GetURI())
	ctx, _ := system.WithSignalShutdown(context.Background())
	go func() {
		err := s.ListenAndServe(ctx)
		assert.NoError(t, err)
	}()
	assert.NoError(t, waitForHealthy(c))

	return NewAPIClient(s.GetURI())
}

func waitForHealthy(c *APIClient) error {
	ch := make(chan bool)
	go func() {
		for {
			healthy, err := c.Healthy()
			if err == nil && healthy {
				ch <- true
				return
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case <-ch:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("server did not reply after 10s")
	}
}


func MakeGenericJob() (*types.JobSpec, *types.JobDeal) {
	return MakeJob(executor.EXECUTOR_DOCKER, verifier.VERIFIER_IPFS)
}

func MakeJob(exec executor.ExecutorType, verif verifier.VerifierType) (*types.JobSpec, *types.JobDeal){
	jobSpec := types.JobSpec{
		Engine:   string(exec),
		Verifier: string(verif),
		Vm: types.JobSpecVm{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"cat",
				"/data/file.txt",
			},
		},
		// Inputs:  inputStorageList,
		// Outputs: testCase.Outputs,
	}

	jobDeal := types.JobDeal{
		Concurrency: 1,
	}

	return &jobSpec, &jobDeal
}
