package test

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/rs/zerolog/log"
)

const TimeToWaitForServerReply = 10 * time.Second
const TimeToWaitForHealthy = 50 * time.Millisecond

func WaitFor(ctx context.Context, c *client.Client, condition func(context.Context, *client.Client) bool) error {
	ch := make(chan bool)
	go func() {
		for {
			if condition(ctx, c) {
				ch <- true
				return
			}
			time.Sleep(TimeToWaitForHealthy)
		}
	}()

	select {
	case <-ch:
		return nil
	case <-time.After(TimeToWaitForServerReply):
		return fmt.Errorf("server did not become alive after %s", TimeToWaitForServerReply)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForAlive waits for the server to be alive
func WaitForAlive(ctx context.Context, c *client.Client) error {
	return WaitFor(ctx, c, func(ctx context.Context, apiClient *client.Client) bool {
		res, err := apiClient.Agent().Alive()
		if err != nil {
			log.Warn().Err(err).Msg("failed to check if server is alive")
			return false
		}
		return res.IsReady()
	})
}

// WaitForNodes waits for the server to be alive and for the node to discover itself
func WaitForNodes(ctx context.Context, c *client.Client) error {
	return WaitFor(ctx, c, func(ctx context.Context, apiClient *client.Client) bool {
		res, err := apiClient.Nodes().List(&apimodels.ListNodesRequest{})
		if err != nil {
			log.Warn().Err(err).Msg("failed to list nodes. retrying...")
			return false
		}
		return len(res.Nodes) > 0
	})
}
