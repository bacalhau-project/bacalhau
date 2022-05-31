package publicapi

import (
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

func TestList(t *testing.T) {
	ipt, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)

	rn, err := requestor_node.NewRequesterNode(ipt)
	assert.NoError(t, err)

	port, err := freeport.GetFreePort()
	assert.NoError(t, err)

	s := NewServer(rn, "0.0.0.0", port)
	c := NewAPIClient(s.GetURI())
	ctx := system.GetCancelContextWithSignals()

	go s.Start(ctx)
	assert.NoError(t, waitForHealthy(c))

	// Should have no jobs initially:
	jobs, err := c.List()
	assert.NoError(t, err)
	assert.Empty(t, jobs)

	// Submit a random job to the node:
	_, err = c.Submit(makeJob())
	assert.NoError(t, err)

	// Should now have one job:
	jobs, err = c.List()
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
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

func makeJob() (*types.JobSpec, *types.JobDeal) {
	jobSpec := types.JobSpec{
		Engine:   string(executor.EXECUTOR_DOCKER),
		Verifier: string(verifier.VERIFIER_IPFS),
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
