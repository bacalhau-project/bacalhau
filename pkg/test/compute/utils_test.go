//go:build integration || !unit

package compute

import (
	"fmt"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
)

func addInput(execution *models.Execution, cid string) *models.Execution {
	execution.Job.Task().InputSources = append(execution.Job.Task().InputSources, &models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceIPFS,
			Params: ipfs.Source{
				CID: cid,
			}.ToMap(),
		},
		Target: "/test_file.txt",
	})
	return execution
}

func addResourceUsage(execution *models.Execution, usage models.Resources) *models.Execution {
	execution.Job.Task().ResourcesConfig = &models.ResourcesConfig{
		CPU:    fmt.Sprintf("%f", usage.CPU),
		Memory: fmt.Sprintf("%d", usage.Memory),
		Disk:   fmt.Sprintf("%d", usage.Disk),
		GPU:    fmt.Sprintf("%d", usage.GPU),
	}
	return execution
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
