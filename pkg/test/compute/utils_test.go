//go:build integration || !unit

package compute

import (
	"context"
	"fmt"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func addInput(execution *models.Execution, sourcePath string) *models.Execution {
	execution.Job.Task().InputSources = append(execution.Job.Task().InputSources, &models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceLocalDirectory,
			Params: localdirectory.Source{
				SourcePath: sourcePath,
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

func getResources(c, m, d string) *models.ResourcesConfig {
	return &models.ResourcesConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

func getResourcesArray(data [][]string) []*models.ResourcesConfig {
	var res []*models.ResourcesConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

type mockInfoProvider struct {
}

func (m mockInfoProvider) GetNodeInfo(ctx context.Context) models.NodeInfo {
	return models.NodeInfo{}
}

func (m mockInfoProvider) RegisterNodeInfoDecorator(decorator models.NodeInfoDecorator) {
}

func (m mockInfoProvider) RegisterLabelProvider(provider models.LabelsProvider) {
}

var _ models.DecoratorNodeInfoProvider = &mockInfoProvider{}
