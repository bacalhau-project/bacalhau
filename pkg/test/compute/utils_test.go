//go:build integration || !unit

package compute

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func generateJob(t testing.TB) model.Job {
	return model.Job{
		Metadata: model.Metadata{
			ID: uuid.New().String(),
		},
		APIVersion: model.APIVersionLatest().String(),
		Spec:       testutils.MakeSpecWithOpts(t),
	}
}

func addInput(job model.Job, cid string) model.Job {
	job.Spec.Inputs = append(job.Spec.Inputs, model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           cid,
		Path:          "/test_file.txt",
	})
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
