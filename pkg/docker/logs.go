package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/moby/moby/pkg/stdcopy"
)

// LogStreamer tails the logs of a container until it exits.
type LogStreamer struct {
	reader io.ReadCloser
	buffer *bytes.Buffer
	cancel context.CancelFunc
	mu     sync.Mutex
}

// Logs returns a copy of the container's logs up until now.
func (ls *LogStreamer) Logs() (string, string, error) { // nolint:gocritic
	ls.mu.Lock()
	defer ls.mu.Unlock()

	buffer := new(bytes.Buffer)
	if _, err := io.Copy(buffer, ls.buffer); err != nil {
		return "", "", fmt.Errorf("failed to copy logs: %w", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if _, err := stdcopy.StdCopy(stdout, stderr, buffer); err != nil {
		return "", "", fmt.Errorf("failed to de-multiplex log streams: %w", err)
	}

	return stdout.String(), stderr.String(), nil
}

// Close cancels the log streamer and releases underlying resources.
func (ls *LogStreamer) Close() {
	ls.cancel()
}

// StreamLogs uses "docker logs --follow" to stream the logs of a container
// until it is stopped. This provides a more robust way of fetching logs from
// an ephemeral container, as the container may be flagged for deletion by the
// daemon before we can poll the logs from it.
// NOTE: If the container is not running, this will exit immediately with the
//       container's current logs (if any). If the container has not been
//       started yet, this will be an empty buffer!
func StreamLogs(ctx context.Context, client *dockerclient.Client, id string) (*LogStreamer, error) {
	cctx, cancel := context.WithCancel(ctx)
	reader, err := client.ContainerLogs(
		cctx,
		id,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to stream logs: %w", err)
	}

	ls := &LogStreamer{
		reader: reader,
		buffer: new(bytes.Buffer),
		cancel: cancel,
	}

	go func() {
		defer reader.Close()
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				buf := make([]byte, 1024) // nolint:gomnd
				n, err := reader.Read(buf)
				if err != nil && err != io.EOF {
					return
				}

				ls.mu.Lock()
				ls.buffer.Write(buf[:n])
				ls.mu.Unlock()

				if err == io.EOF {
					return
				}
			}
		}
	}()

	return ls, nil
}
