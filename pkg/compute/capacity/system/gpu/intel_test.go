//go:build unit || !integration

package gpu

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/require"
)

//go:embed xpu-smi-discovery-output-many.json
var manyListOutput []byte

//go:embed xpu-smi-discovery-output-one.json
var oneListOutput []byte

//go:embed xpu-smi-discovery-device-output.json
var infoOutput []byte

// fakeToolProvider wraps another tool provider and just calls its Parser with
// fake input.
type fakeToolProvider struct {
	Provider  *capacity.ToolBasedProvider
	FakeInput io.Reader
}

// GetAvailableCapacity implements capacity.Provider.
func (fake *fakeToolProvider) GetAvailableCapacity(ctx context.Context) (models.Resources, error) {
	return fake.Provider.Parser(fake.FakeInput)
}

// ResourceTypes implements capacity.Provider.
func (fake *fakeToolProvider) ResourceTypes() []string {
	return fake.Provider.ResourceTypes()
}

var _ capacity.Provider = (*fakeToolProvider)(nil)

func getTestProvider(listToolOutput, infoToolOutput []byte) *intelGPUProvider {
	return &intelGPUProvider{
		listProvider: &fakeToolProvider{Provider: &xpuDeviceListProvider, FakeInput: bytes.NewReader(listToolOutput)},
		getInfoProvider: func(deviceId int) capacity.Provider {
			return &fakeToolProvider{Provider: &xpuDeviceInfoProvider, FakeInput: bytes.NewReader(infoToolOutput)}
		},
	}
}

func TestParsingIntelGPUsWithNone(t *testing.T) {
	provider := getTestProvider(
		[]byte(`{"device_list":[]}`),
		[]byte(`{}`),
	)
	output, err := provider.GetAvailableCapacity(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(0), output.GPU)
	require.Empty(t, output.GPUs)
}

func TestParsingIntelGPUsWithOne(t *testing.T) {
	provider := getTestProvider(oneListOutput, infoOutput)
	output, err := provider.GetAvailableCapacity(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(1), output.GPU)
	require.Len(t, output.GPUs, 1)

	gpu := output.GPUs[0]
	require.Equal(t, models.GPUVendorIntel, gpu.Vendor)
	require.Equal(t, uint64(0), gpu.Index)
	require.Equal(t, "0000:e9:00.0", gpu.PCIAddress)
	require.Equal(t, "Intel Corporation Device 56c1 (rev 05)", gpu.Name)
	require.Equal(t, uint64(5068), gpu.Memory)

}

func TestParsingIntelGPUsWithMany(t *testing.T) {
	provider := getTestProvider(manyListOutput, infoOutput)
	output, err := provider.GetAvailableCapacity(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(2), output.GPU)
	require.Len(t, output.GPUs, 2)

	for _, gpu := range output.GPUs {
		require.Equal(t, models.GPUVendorIntel, gpu.Vendor)
		require.Equal(t, uint64(0), gpu.Index)
		require.Equal(t, "0000:e9:00.0", gpu.PCIAddress)
		require.Equal(t, "Intel Corporation Device 56c1 (rev 05)", gpu.Name)
		require.Equal(t, uint64(5068), gpu.Memory)
	}
}
