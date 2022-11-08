package computenode

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

func GetJobSpec(cid string) model.Spec {
	inputs := []model.StorageSpec{}
	if cid != "" {
		inputs = []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				CID:           cid,
				Path:          "/test_file.txt",
			},
		}
	}
	return model.Spec{
		Engine:   model.EngineNoop,
		Verifier: model.VerifierNoop,
		Inputs:   inputs,
	}
}

func GetProbeData(cid string) computenode.JobSelectionPolicyProbeData {
	return computenode.JobSelectionPolicyProbeData{
		NodeID: "test",
		JobID:  "test",
		Spec:   GetJobSpec(cid),
	}
}

//nolint:unused
func getResources(c, m, d string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused
func getResourcesArray(data [][]string) []model.ResourceUsageConfig {
	var res []model.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

func RunJobGetStdout(
	ctx context.Context,
	t *testing.T,
	computeNode *computenode.ComputeNode,
	spec model.Spec,
) string {
	result := t.TempDir()

	j := &model.Job{
		ID:   "test",
		Spec: spec,
	}
	shard := model.JobShard{
		Job:   j,
		Index: 0,
	}
	runnerOutput, err := computeNode.RunShardExecution(ctx, shard, result)
	require.NoError(t, err)
	require.Empty(t, runnerOutput.ErrorMsg)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(t, result, "The job result folder exists")
	require.FileExists(t, stdoutPath, "The stdout file exists")
	dat, err := os.ReadFile(stdoutPath)
	require.NoError(t, err)
	return string(dat)
}
