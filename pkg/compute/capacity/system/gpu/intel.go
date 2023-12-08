package gpu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/pkg/errors"
)

// https://github.com/intel/xpumanager/blob/251edc28b27c8af7dceea18154d5f8fb170fc9df/doc/smi_user_guide.md

type xpuDeviceList struct {
	List []struct {
		DeviceID   uint64 `json:"device_id"`
		DeviceName string `json:"device_name"`
	} `json:"device_list"`
}

var xpuDeviceListProvider = capacity.ToolBasedProvider{
	Command:  "xpu-smi",
	Provides: "Intel GPUs",
	Args:     []string{"discovery", "--json"},
	Parser: func(output io.Reader) (models.Resources, error) {
		var records xpuDeviceList
		err := json.NewDecoder(output).Decode(&records)
		if err != nil {
			return models.Resources{}, err
		}

		gpus := make([]models.GPU, len(records.List))
		for index, record := range records.List {
			gpus[index].Index = record.DeviceID
			gpus[index].Name = record.DeviceName
			gpus[index].Vendor = models.GPUVendorIntel
			gpus[index].PCIAddress = record.PCIAddress
		}
		return models.Resources{GPU: uint64(len(gpus)), GPUs: gpus}, nil
	},
}

type xpuDeviceInfo struct {
	DeviceID    uint64 `json:"device_id"`
	DeviceName  string `json:"device_name"`
	TotalMemory string `json:"memory_physical_size_byte"`
}

var xpuDeviceInfoProvider = capacity.ToolBasedProvider{
	Command:  "xpu-smi",
	Provides: "Intel GPUs",
	// note: Args require a device ID, appended later
	Args:     []string{"discovery", "--json", "--device"},
	Parser: func(output io.Reader) (models.Resources, error) {
		var record xpuDeviceInfo
		err := json.NewDecoder(output).Decode(&record)
		if err != nil {
			return models.Resources{}, err
		}

		parsedMemoryBytes, err := strconv.ParseUint(record.TotalMemory, 10, 64)
		if err != nil {
			return models.Resources{}, errors.Wrap(err, "error parsing memory")
		}

		gpu := models.GPU{
			Index:      record.DeviceID,
			Name:       record.DeviceName,
			Vendor: models.GPUVendorIntel,
			Memory: parsedMemoryBytes / bytesPerMebibyte,
		}

		return models.Resources{GPU: 1, GPUs: []models.GPU{gpu}}, nil
	},
}

type intelGPUProvider struct {
	listProvider    capacity.Provider
	getInfoProvider func(deviceId int) capacity.Provider
}

func NewIntelGPUProvider() capacity.Provider {
	return &intelGPUProvider{
		listProvider: &xpuDeviceListProvider,
		getInfoProvider: func(deviceId int) capacity.Provider {
			provider := xpuDeviceInfoProvider
			provider.Args = append(provider.Args, fmt.Sprint(deviceId))
			return &provider
		},
	}
}

func (intel *intelGPUProvider) GetAvailableCapacity(ctx context.Context) (models.Resources, error) {
	// First get the list of Intel GPUs on the system.
	gpuList, err := intel.listProvider.GetAvailableCapacity(ctx)
	if err != nil {
		return models.Resources{}, err
	}

	// Now look up each GPU individually to get its memory information
	// Start with an empty Resources and just fold over it
	var allGPUs models.Resources
	for _, gpu := range gpuList.GPUs {
		provider := intel.getInfoProvider(int(gpu.Index))
		gpuInfo, err := provider.GetAvailableCapacity(ctx)
		if err != nil {
			return models.Resources{}, err
		}

		allGPUs = *allGPUs.Add(gpuInfo)
	}

	return allGPUs, nil
}

func (*intelGPUProvider) ResourceTypes() []string {
	return xpuDeviceInfoProvider.ResourceTypes()
}

var _ capacity.Provider = (*intelGPUProvider)(nil)
