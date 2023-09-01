package test

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
)

const TimeToWaitForServerReply = 10 * time.Second
const TimeToWaitForHealthy = 50 * time.Millisecond

func WaitFor(ctx context.Context, c *client.APIClient, condition func(context.Context, *client.APIClient) bool) error {
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
		return fmt.Errorf("server did not become heal after %s", TimeToWaitForServerReply)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForAlive waits for the server to be alive
func WaitForAlive(ctx context.Context, c *client.APIClient) error {
	return WaitFor(ctx, c, func(ctx context.Context, apiClient *client.APIClient) bool {
		alive, err := apiClient.Alive(ctx)
		if err != nil || !alive {
			return false
		}
		return true
	})
}

// WaitForNodes waits for the server to be alive and for the node to discover itself
func WaitForNodes(ctx context.Context, c *client.APIClient) error {
	return WaitFor(ctx, c, func(ctx context.Context, apiClient *client.APIClient) bool {
		nodes, err := apiClient.Nodes(ctx)
		if err != nil || len(nodes) == 0 {
			return false
		}
		return true
	})
}
