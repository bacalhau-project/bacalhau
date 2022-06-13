package publicapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
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
			healthy, err := c.Healthy(context.Background())
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
