package publicapi

import (
	"context"
	"fmt"
	"time"
)

const TimeToWaitForServerReply = 10 * time.Second
const TimeToWaitForHealthy = 50 * time.Millisecond

// WaitForHealthy waits for the server to be alive and for the node to discover itself
func WaitForHealthy(ctx context.Context, c *RequesterAPIClient) error {
	ch := make(chan bool)
	go func() {
		for {
			// wait for the server to be alive and for the node to discover itself
			alive, err := c.Alive(ctx)
			if err == nil && alive {
				nodes, err := c.Nodes(ctx)
				if err == nil && len(nodes) > 0 {
					ch <- true
					return
				}
			}

			time.Sleep(TimeToWaitForHealthy)
		}
	}()

	select {
	case <-ch:
		return nil
	case <-time.After(TimeToWaitForServerReply):
		return fmt.Errorf("server did not become heal after %s", TimeToWaitForServerReply)
	}
}
