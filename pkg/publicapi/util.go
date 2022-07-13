package publicapi

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/requestornode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_utils "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

var TimeToWaitForServerReply = 10 // nolint:gomnd // magic number appropriate here
var TimeToWaitForHealthy = 50     // nolint:gomnd // magic number appropriate here

// SetupTests sets up a client for a requester node's API server, for testing.
func SetupTests(t *testing.T) (*APIClient, *system.CleanupManager) {
	system.InitConfigForTesting(t)

	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTracer)

	ipt, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	noopVerifiers, err := verifier_utils.NewNoopVerifiers(cm)
	require.NoError(t, err)

	rn, err := requestornode.NewRequesterNode(
		cm,
		ipt,
		noopVerifiers,
	)
	require.NoError(t, err)

	host := "0.0.0.0"
	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	s := NewServer(rn, host, port, ipt)
	c := NewAPIClient(s.GetURI())
	go func() {
		require.NoError(t, s.ListenAndServe(context.Background(), cm))
	}()
	require.NoError(t, waitForHealthy(c))

	return NewAPIClient(s.GetURI()), cm
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

			time.Sleep(time.Duration(TimeToWaitForHealthy) * time.Millisecond)
		}
	}()

	select {
	case <-ch:
		return nil
	case <-time.After(time.Duration(TimeToWaitForServerReply) * time.Second):
		return fmt.Errorf("server did not reply after %ss", time.Duration(TimeToWaitForServerReply)*time.Second)
	}
}

// Function to get disk usage of path/disk
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
	c := exec.Command("tail", strconv.Itoa(count), path) // nolint:gosec // subprocess not at risk
	output, err := c.Output()
	if err != nil {
		log.Warn().Msgf("Could not find file at %s", path)
		return nil, err
	}
	return output, nil
}

func MakeGenericJob() (executor.JobSpec, executor.JobDeal) {
	return MakeJob(executor.EngineDocker, verifier.VerifierIpfs)
}

func MakeNoopJob() (executor.JobSpec, executor.JobDeal) {
	return MakeJob(executor.EngineNoop, verifier.VerifierIpfs)
}

func MakeJob(engineType executor.EngineType, verifierType verifier.VerifierType) (executor.JobSpec, executor.JobDeal) {
	jobSpec := executor.JobSpec{
		Engine:   engineType,
		Verifier: verifierType,
		Docker: executor.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"cat",
				"/data/file.txt",
			},
		},
		// Inputs:  inputStorageList,
		// Outputs: testCase.Outputs,
	}

	jobDeal := executor.JobDeal{
		Concurrency: 1,
	}

	return jobSpec, jobDeal
}
