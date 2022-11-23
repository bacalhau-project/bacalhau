package compute

import (
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

func GetJob(cid string) *model.Job {
	return &model.Job{
		ID:   "test",
		Spec: GetJobSpec(cid),
	}
}

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
