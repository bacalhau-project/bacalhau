package publicapi

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	publisher_utils "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_utils "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

const TimeToWaitForServerReply = 10
const TimeToWaitForHealthy = 50

// SetupTests sets up a client for a requester node's API server, for testing.
func SetupTests(t *testing.T) (*APIClient, *system.CleanupManager) {
	system.InitConfigForTesting(t)

	cleanupManager := system.NewCleanupManager()
	cleanupManager.RegisterCallback(system.CleanupTracer)

	inprocessTransport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	inmemoryDatastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	noopStorageProviders, err := util.NewNoopStorageProviders(cleanupManager)
	require.NoError(t, err)

	c, err := controller.NewController(
		cleanupManager,
		inmemoryDatastore,
		inprocessTransport,
		noopStorageProviders,
	)
	require.NoError(t, err)

	noopPublishers, err := publisher_utils.NewNoopPublishers(
		cleanupManager,
		c.GetStateResolver(),
	)
	require.NoError(t, err)

	noopVerifiers, err := verifier_utils.NewNoopVerifiers(
		cleanupManager,
		c.GetStateResolver(),
	)
	require.NoError(t, err)

	_, err = requesternode.NewRequesterNode(
		cleanupManager,
		c,
		noopVerifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	host := "0.0.0.0"
	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	s := NewServer(host, port, c, noopPublishers)
	cl := NewAPIClient(s.GetURI())
	go func() {
		require.NoError(t, s.ListenAndServe(context.Background(), cleanupManager))
	}()
	require.NoError(t, waitForHealthy(cl))

	return NewAPIClient(s.GetURI()), cleanupManager
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
	c := exec.Command("tail", strconv.Itoa(count), path) //nolint:gosec // subprocess not at risk
	output, err := c.Output()
	if err != nil {
		log.Warn().Msgf("Could not find file at %s", path)
		return nil, err
	}
	return output, nil
}

func MakeEchoJob() (executor.JobSpec, executor.JobDeal) {
	randomSuffix, _ := uuid.NewUUID()
	return MakeJob(executor.EngineDocker, verifier.VerifierNoop, []string{
		"echo",
		randomSuffix.String(),
	})
}

func MakeGenericJob() (executor.JobSpec, executor.JobDeal) {
	return MakeJob(executor.EngineDocker, verifier.VerifierNoop, []string{
		"cat",
		"/data/file.txt",
	})
}

func MakeNoopJob() (executor.JobSpec, executor.JobDeal) {
	return MakeJob(executor.EngineNoop, verifier.VerifierNoop, []string{
		"cat",
		"/data/file.txt",
	})
}

func MakeJob(engineType executor.EngineType,
	verifierType verifier.VerifierType,
	entrypointArray []string) (executor.JobSpec, executor.JobDeal) {
	jobSpec := executor.JobSpec{
		Engine:   engineType,
		Verifier: verifierType,
		Docker: executor.JobSpecDocker{
			Image:      "ubuntu:latest",
			Entrypoint: entrypointArray,
		},
		// Inputs:  inputStorageList,
		// Outputs: testCase.Outputs,
	}

	jobDeal := executor.JobDeal{
		Concurrency: 1,
	}

	return jobSpec, jobDeal
}
