package publicapi

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	publisher_utils "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	verifier_utils "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

const TimeToWaitForServerReply = 10
const TimeToWaitForHealthy = 50

// SetupTests sets up a client for a requester node's API server, for testing.
func SetupTests(t *testing.T) (*APIClient, *system.CleanupManager) {
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	return SetupTestsWithPort(t, port)
}

func SetupTestsWithPort(t *testing.T, port int) (*APIClient, *system.CleanupManager) {
	// Setup the system
	err := system.InitConfigForTesting()
	require.NoError(t, err)

	cm := system.NewCleanupManager()
	ctx := context.Background()

	cm.RegisterCallback(system.CleanupTraceProvider)

	inprocessTransport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	inmemoryDatastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	noopStorageProviders, err := util.NewNoopStorageProviders(ctx, cm, noop_storage.StorageConfig{})
	require.NoError(t, err)

	c, err := controller.NewController(
		ctx,
		cm,
		inmemoryDatastore,
		inprocessTransport,
		noopStorageProviders,
	)
	require.NoError(t, err)

	noopPublishers, err := publisher_utils.NewNoopPublishers(
		ctx,
		cm,
		c.GetStateResolver(),
	)
	require.NoError(t, err)

	noopVerifiers, err := verifier_utils.NewNoopVerifiers(
		ctx,
		cm,
		c.GetStateResolver(),
	)
	require.NoError(t, err)

	_, err = requesternode.NewRequesterNode(
		ctx,
		cm,
		c,
		noopVerifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	host := "0.0.0.0"

	s := NewServer(ctx, host, port, c, noopPublishers)
	cl := NewAPIClient(s.GetURI())
	go func() {
		require.NoError(t, s.ListenAndServe(ctx, cm))
	}()
	require.NoError(t, waitForHealthy(ctx, cl))

	return NewAPIClient(s.GetURI()), cm
}

func waitForHealthy(ctx context.Context, c *APIClient) error {
	ch := make(chan bool)
	go func() {
		for {
			alive, err := c.Alive(ctx)
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
	usage := du.NewDiskUsage(path)
	if usage == nil {
		return
	}
	disk.All = usage.Size()
	disk.Free = usage.Free()
	disk.Used = usage.Used()
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

func MakeEchoJob() *model.Job {
	randomSuffix, _ := uuid.NewUUID()
	return MakeJob(model.EngineDocker, model.VerifierNoop, model.PublisherNoop, []string{
		"echo",
		randomSuffix.String(),
	})
}

func MakeGenericJob() *model.Job {
	return MakeJob(model.EngineDocker, model.VerifierNoop, model.PublisherNoop, []string{
		"echo",
		"$(date +%s)",
	})
}

func MakeNoopJob() *model.Job {
	return MakeJob(model.EngineNoop, model.VerifierNoop, model.PublisherNoop, []string{
		"echo",
		"$(date +%s)",
	})
}

func MakeJob(
	engineType model.Engine,
	verifierType model.VerifierType,
	publisherType model.PublisherType,
	entrypointArray []string) *model.Job {
	j := &model.Job{}

	j.Spec = model.Spec{
		Engine:    engineType,
		Verifier:  verifierType,
		Publisher: publisherType,
		Docker: model.JobSpecDocker{
			Image:      "ubuntu:latest",
			Entrypoint: entrypointArray,
		},
		// Inputs:  inputStorageList,
		// Outputs: testCase.Outputs,
	}

	j.Deal = model.Deal{
		Concurrency: 1,
	}

	return j
}
