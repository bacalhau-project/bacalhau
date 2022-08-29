package computenode

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/require"
)



func GetJobSpec(cid string) executor.JobSpec {
	inputs := []storage.StorageSpec{}
	if cid != "" {
		inputs = []model.StorageSpec{
			{
				Engine: model.StorageSourceIPFS,
				Cid:    cid,
				Path:   "/test_file.txt",
			},
		}
	}
	return model.JobSpec{
		Engine:   model.EngineDocker,
		Verifier: model.VerifierNoop,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"cat",
				"/test_file.txt",
			},
		},
		Inputs: inputs,
	}
}

func GetProbeData(cid string) computenode.JobSelectionPolicyProbeData {
	return computenode.JobSelectionPolicyProbeData{
		NodeID: "test",
		JobID:  "test",
		Spec:   GetJobSpec(cid),
	}
}

//nolint:unused,deadcode
func getResources(c, m, d string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused,deadcode
func getResourcesArray(data [][]string) []model.ResourceUsageConfig {
	var res []model.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

func RunJobGetStdout(
	t *testing.T,
	ctx context.Context,
	computeNode *computenode.ComputeNode,
	spec model.JobSpec,
) string {
	result, err := ioutil.TempDir("", "bacalhau-RunJobGetStdout")
	require.NoError(t, err)

	job := model.Job{
		ID:   "test",
		Spec: spec,
	}
	shard := model.JobShard{
		Job:   job,
		Index: 0,
	}
	err = computeNode.RunShardExecution(ctx, shard, result)
	require.NoError(t, err)

	stdoutPath := fmt.Sprintf("%s/stdout", result)
	require.DirExists(t, result, "The job result folder exists")
	require.FileExists(t, stdoutPath, "The stdout file exists")
	dat, err := os.ReadFile(stdoutPath)
	require.NoError(t, err)
	return string(dat)
}
