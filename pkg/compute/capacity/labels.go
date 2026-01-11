package capacity

import (
	"context"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type gpuLabelsProvider struct {
	resources models.Resources
}

func NewGPULabelsProvider(totalCapacity models.Resources) models.LabelsProvider {
	return gpuLabelsProvider{resources: totalCapacity}
}

// GetLabels implements models.LabelsProvider.
func (p gpuLabelsProvider) GetLabels(ctx context.Context) map[string]string {
	labels := make(map[string]string, len(p.resources.GPUs)*2)
	for i, gpu := range p.resources.GPUs {
		// Model label e.g. GPU-0: Tesla-T1
		key := fmt.Sprintf("GPU-%d", i)
		name := strings.ReplaceAll(gpu.Name, " ", "-") // Replace spaces with dashes
		labels[key] = name

		// Memory label e.g. GPU-0-Memory: 15360-MiB
		key = fmt.Sprintf("GPU-%d-Memory", i)
		memory := strings.ReplaceAll(fmt.Sprintf("%d MiB", gpu.Memory), " ", "-") // Replace spaces with dashes
		labels[key] = memory
	}
	return labels
}
