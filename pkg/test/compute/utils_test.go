package compute

import (
	"fmt"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/google/uuid"
)

func generateJob() model.Job {
	return model.Job{
		ID:         uuid.New().String(),
		APIVersion: model.APIVersionLatest().String(),
		Spec: model.Spec{
			Engine:    model.EngineNoop,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherNoop,
		},
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
