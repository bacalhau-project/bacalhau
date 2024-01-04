package gpu

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// TODO(forrest) consider switching to: https://github.com/NVIDIA/gpu-monitoring-tools
const (
	// nvidiaCLI is the path to the Nvidia helper binary
	nvidiaCLI = "nvidia-smi"
	// nvidiaCLIArgs is the args we pass the nvidiaCLI
	nvidiaCLIQueryArg  = "--query-gpu=index,gpu_name,memory.total"
	nvidiaCLIFormatArg = "--format=csv,noheader,nounits"
)

func NewNvidiaGPUProvider() *capacity.ToolBasedProvider {
	return &capacity.ToolBasedProvider{
		Command:  nvidiaCLI,
		Provides: "Nvidia GPUs",
		Args:     []string{nvidiaCLIQueryArg, nvidiaCLIFormatArg},
		Parser:   parseNvidiaCliOutput,
	}
}

func parseNvidiaCliOutput(resp io.Reader) (models.Resources, error) {
	reader := csv.NewReader(resp)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return models.Resources{}, err
	}

	gpus := make([]models.GPU, len(records))
	for index, record := range records {
		gpus[index].Index, err = strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			return models.Resources{}, err
		}

		gpus[index].Vendor = models.GPUVendorNvidia
		gpus[index].Name = record[1]
		gpus[index].Memory, err = strconv.ParseUint(record[2], 10, 64)
		if err != nil {
			return models.Resources{}, err
		}
	}

	return models.Resources{GPU: uint64(len(gpus)), GPUs: gpus}, nil
}
