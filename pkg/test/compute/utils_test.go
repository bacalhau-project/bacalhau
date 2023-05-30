//go:build integration || !unit

package compute

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
)

func generateJob(t testing.TB) model.Job {
	return model.Job{
		Metadata: model.Metadata{
			ID: uuid.New().String(),
		},
		APIVersion: model.APIVersionLatest().String(),
		Spec: model.Spec{
			Engine:   enginetesting.NoopMakeEngine(t, "noop"),
			Verifier: model.VerifierNoop,
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherNoop,
			},
		},
	}
}

func addInput(t testing.TB, job model.Job, cid cid.Cid) model.Job {
	ipfsspec, err := (&ipfs.IPFSStorageSpec{CID: cid}).AsSpec("TODO", "/test_file.txt")
	require.NoError(t, err)
	job.Spec.Inputs = append(job.Spec.Inputs, ipfsspec)
	return job
}

func addResourceUsage(job model.Job, usage model.ResourceUsageData) model.Job {
	job.Spec.Resources = model.ResourceUsageConfig{
		CPU:    fmt.Sprintf("%f", usage.CPU),
		Memory: fmt.Sprintf("%d", usage.Memory),
		Disk:   fmt.Sprintf("%d", usage.Disk),
		GPU:    fmt.Sprintf("%d", usage.GPU),
	}
	return job
}

func getResources(c, m, d string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

func getResourcesArray(data [][]string) []model.ResourceUsageConfig {
	var res []model.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}
