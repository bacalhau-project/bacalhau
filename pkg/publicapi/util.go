package publicapi

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	publisher_utils "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	verifier_utils "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/phayes/freeport"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

const TimeToWaitForServerReply = 10
const TimeToWaitForHealthy = 50

// SetupRequesterNodeForTests sets up a client for a requester node's API server, for testing.
func SetupRequesterNodeForTests(t *testing.T, hairpin bool) (*APIClient, *system.CleanupManager) {
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	return SetupRequesterNodeForTestWithPort(t, port, hairpin)
}

func SetupRequesterNodeForTestWithPort(t *testing.T, port int, hairpin bool) (*APIClient, *system.CleanupManager) {
	return SetupRequesterNodeForTestsWithPortAndConfig(t, port, DefaultAPIServerConfig, hairpin)
}

func SetupRequesterNodeForTestsWithConfig(t *testing.T, config *APIServerConfig, hairpin bool) (*APIClient, *system.CleanupManager) {
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	return SetupRequesterNodeForTestsWithPortAndConfig(t, port, config, hairpin)
}

// TODO: we are almost establishing a full node to test the API. Most of these tests should be move to test package,
// and only keep simple unit tests here.
//
//nolint:funlen
func SetupRequesterNodeForTestsWithPortAndConfig(
	t *testing.T, port int, config *APIServerConfig, hairpin bool,
) (*APIClient, *system.CleanupManager) {
	// Setup the system
	err := system.InitConfigForTesting(t)
	require.NoError(t, err)

	cm := system.NewCleanupManager()
	ctx := context.Background()

	cm.RegisterCallback(system.CleanupTraceProvider)

	inprocessTransport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	inmemoryDatastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	noopStorageProviders, err := util.NewNoopStorageProvider(ctx, cm, noop_storage.StorageConfig{})
	require.NoError(t, err)

	noopPublishers, err := publisher_utils.NewNoopPublishers(
		ctx,
		cm,
		localdb.GetStateResolver(inmemoryDatastore),
	)
	require.NoError(t, err)

	noopVerifiers, err := verifier_utils.NewNoopVerifiers(
		ctx,
		cm,
		localdb.GetStateResolver(inmemoryDatastore),
	)
	require.NoError(t, err)

	// prepare event handlers
	tracerContextProvider := system.NewTracerContextProvider(inprocessTransport.HostID())
	noopContextProvider := system.NewNoopContextProvider()
	cm.RegisterCallback(tracerContextProvider.Shutdown)

	localEventConsumer := eventhandler.NewChainedLocalEventHandler(noopContextProvider)
	jobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)
	jobEventPublisher := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	requesterNode, err := requesternode.NewRequesterNode(
		ctx,
		cm,
		inprocessTransport.HostID(),
		inmemoryDatastore,
		localEventConsumer,
		jobEventPublisher,
		noopVerifiers,
		noopStorageProviders,
		requesternode.NewDefaultRequesterNodeConfig(),
	)
	require.NoError(t, err)

	localDBEventHandler := localdb.NewLocalDBEventHandler(inmemoryDatastore)

	host := "0.0.0.0"
	s := NewServerWithConfig(ctx, host, port, inmemoryDatastore, inprocessTransport,
		requesterNode, []model.DebugInfoProvider{}, noopPublishers, noopStorageProviders, config)

	// order of event handlers is important as triggering some handlers should depend on the state of others.
	jobEventConsumer.AddHandlers(
		tracerContextProvider,
		localDBEventHandler,
		requesterNode,
		s, // websockets
	)

	jobEventPublisher.AddHandlers(
		eventhandler.JobEventHandlerFunc(inprocessTransport.Publish),
	)

	localEventConsumer.AddHandlers(
		localDBEventHandler,
	)

	// Do we send events that we emit back to ourselves? Needed for test
	// compatibility. This _does_ happen in production. See node.go
	if hairpin {
		inprocessTransport.Subscribe(ctx, jobEventConsumer.HandleJobEvent)
	}

	cl := NewAPIClient(s.GetURI())
	wait := make(chan struct{})
	go func() {
		defer func() {
			wait <- struct{}{}
		}()
		require.NoError(t, s.ListenAndServe(ctx, cm))
	}()
	cm.RegisterCallback(func() error {
		// This ensures that the test only ends _after_ the API client has finished closing down,
		// so we avoid panics from attempting to log to testing.T after the test has closed
		<-wait
		return nil
	})
	require.NoError(t, waitForHealthy(ctx, cl))

	return cl, cm
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
	verifierType model.Verifier,
	publisherType model.Publisher,
	entrypointArray []string) *model.Job {
	j := model.NewJob()

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
