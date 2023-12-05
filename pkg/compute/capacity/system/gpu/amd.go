package gpu

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const rocmCommand = "rocm-smi"
const bytesPerMebibyte = 1048576

var rocmArgs = []string{"--showproductname", "--showmeminfo", "vram", "--json"}

// {"card0": {"VRAM Total Memory (B)": "68702699520", "VRAM Total Used Memory
// (B)": "10960896", "Card series": "Instinct MI210", "Card model": "0x0c34",
// "Card vendor": "Advanced Micro Devices, Inc. [AMD/ATI]", "Card SKU":
// "D67301"}}
type rocmGPU struct {
	TotalMemory string `json:"VRAM Total Memory (B)"`
	UsedMemory  string `json:"VRAM Total Used Memory (B)"`
	Series      string `json:"Card series"`
	Model       string `json:"Card model"`
	Vendor      string `json:"Card vendor"`
	SKU         string `json:"Card SKU"`
}

type rocmGPUList map[string]rocmGPU

func parseRocmSMIOutput(output io.Reader) (models.Resources, error) {
	var records rocmGPUList
	err := json.NewDecoder(output).Decode(&records)
	if err != nil {
		return models.Resources{}, err
	}

	gpus := make([]models.GPU, len(records))
	for cardNumber, record := range records {
		index, err := strconv.ParseUint(cardNumber[4:], 10, 64)
		if err != nil {
			return models.Resources{}, err
		}
		gpus[index].Index = index
		gpus[index].Name = record.Series
		memBytes, err := strconv.ParseUint(record.TotalMemory, 10, 64)
		if err != nil {
			return models.Resources{}, err
		}
		gpus[index].Memory = memBytes / bytesPerMebibyte // convert to mebibytes
		gpus[index].Vendor = models.GPUVendorAMDATI
	}

	return models.Resources{GPU: uint64(len(gpus)), GPUs: gpus}, nil
}

func NewAMDGPUProvider() *capacity.ToolBasedProvider {
	return &capacity.ToolBasedProvider{
		Command:  rocmCommand,
		Provides: "AMD GPUs",
		Args:     rocmArgs,
		Parser:   parseRocmSMIOutput,
	}
}
