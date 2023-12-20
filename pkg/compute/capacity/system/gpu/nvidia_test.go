//go:build unit || !integration

package gpu

import (
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestParsingNvidiaGPUsWithMany(t *testing.T) {
	output := strings.Join([]string{
		"0, Tesla T4, 15360",
		"1, Tesla T1, 12345",
	}, "\n")

	resources, err := parseNvidiaCliOutput(strings.NewReader(output))
	require.Equal(t, int(resources.GPU), len(resources.GPUs))
	gpus := resources.GPUs
	require.NoError(t, err)
	require.Len(t, gpus, 2)
	require.Equal(t, uint64(0), gpus[0].Index)
	require.Equal(t, "Tesla T4", gpus[0].Name)
	require.Equal(t, uint64(15360), gpus[0].Memory)
	require.Equal(t, models.GPUVendorNvidia, gpus[0].Vendor)
	require.Equal(t, uint64(1), gpus[1].Index)
	require.Equal(t, "Tesla T1", gpus[1].Name)
	require.Equal(t, uint64(12345), gpus[1].Memory)
	require.Equal(t, models.GPUVendorNvidia, gpus[1].Vendor)
}

func TestParsingNvidiaGPUsWithOne(t *testing.T) {
	output := strings.Join([]string{
		"0, Tesla T4, 15360",
	}, "\n")

	resources, err := parseNvidiaCliOutput(strings.NewReader(output))
	require.Equal(t, int(resources.GPU), len(resources.GPUs))
	gpus := resources.GPUs
	require.NoError(t, err)
	require.Len(t, gpus, 1)
	require.Equal(t, uint64(0), gpus[0].Index)
	require.Equal(t, "Tesla T4", gpus[0].Name)
	require.Equal(t, uint64(15360), gpus[0].Memory)
	require.Equal(t, models.GPUVendorNvidia, gpus[0].Vendor)
}

func TestParsingNvidiaGPUsWithNone(t *testing.T) {
	output := strings.Join([]string{}, "\n")

	resources, err := parseNvidiaCliOutput(strings.NewReader(output))
	require.Equal(t, int(resources.GPU), len(resources.GPUs))
	gpus := resources.GPUs
	require.NoError(t, err)
	require.Len(t, gpus, 0)
}
