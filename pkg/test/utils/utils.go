package testutils

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetJobFromTestOutput(ctx context.Context, t *testing.T, c *publicapi.RequesterAPIClient, out string) model.Job {
	jobID := system.FindJobIDInTestOutput(out)
	uuidRegex := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	require.Regexp(t, uuidRegex, jobID, "Job ID should be a UUID")

	j, _, err := c.Get(ctx, jobID)
	require.NoError(t, err)
	require.NotNil(t, j, "Failed to get job with ID: %s", out)
	return j.Job
}

func FirstFatalError(_ *testing.T, output string) (model.TestFatalErrorHandlerContents, error) {
	linesInOutput := system.SplitLines(output)
	fakeFatalError := &model.TestFatalErrorHandlerContents{}
	for _, line := range linesInOutput {
		err := model.JSONUnmarshalWithMax([]byte(line), fakeFatalError)
		if err != nil {
			return model.TestFatalErrorHandlerContents{}, err
		} else {
			return *fakeFatalError, nil
		}
	}
	return model.TestFatalErrorHandlerContents{}, fmt.Errorf("no fatal error found in output")
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
		EngineDeprecated: engineType,
		Verifier:         verifierType,
		PublisherSpec: model.PublisherSpec{
			Type: publisherType,
		},
		EngineSpec: model.NewDockerEngineSpec("ubuntu:latest", entrypointArray, nil, ""),
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}

	return j
}

// WaitForNodeDiscovery for the requester node to pick up the nodeInfo messages
func WaitForNodeDiscovery(t *testing.T, requesterNode *node.Node, expectedNodeCount int) {
	ctx := context.Background()
	waitDuration := 15 * time.Second
	waitGaps := 20 * time.Millisecond
	waitUntil := time.Now().Add(waitDuration)
	loggingGap := 1 * time.Second
	waitLoggingUntil := time.Now().Add(loggingGap)

	var nodeInfos []model.NodeInfo
	for time.Now().Before(waitUntil) {
		var err error
		nodeInfos, err = requesterNode.NodeInfoStore.List(ctx)
		require.NoError(t, err)
		if time.Now().After(waitLoggingUntil) {
			t.Logf("connected to %d peers: %v", len(nodeInfos), logger.ToSliceStringer(nodeInfos, func(t model.NodeInfo) string {
				return t.PeerInfo.ID.String()
			}))
			waitLoggingUntil = time.Now().Add(loggingGap)
		}
		if len(nodeInfos) == expectedNodeCount {
			return
		}
		time.Sleep(waitGaps)
	}
	require.FailNowf(t, fmt.Sprintf("requester node didn't read all node infos even after waiting for %s", waitDuration),
		"expected 4 node infos, got %d. %+v", len(nodeInfos), logger.ToSliceStringer(nodeInfos, func(t model.NodeInfo) string {
			return t.PeerInfo.ID.String()
		}))
}
