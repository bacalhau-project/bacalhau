package publicapi

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
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
			alive, err := c.Alive()
			if err == nil && alive {
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

// Function to get
// disk usage of path/disk
func MountUsage(path string) (disk types.MountStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks * uint64(fs.Bsize)
	disk.Free = fs.Bfree * uint64(fs.Bsize)
	disk.Used = disk.All - disk.Free
	return
}

// Defining constants to convert size units
const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

// use "-1" as count for just last line
func TailFile(count int, path string) ([]byte, error) {
	c := exec.Command("tail", strconv.Itoa(count), path)
	output, err := c.Output()
	if err != nil {
		log.Warn().Msgf("Could not find file at %s", path)
		return nil, err
	}
	return output, nil
}
