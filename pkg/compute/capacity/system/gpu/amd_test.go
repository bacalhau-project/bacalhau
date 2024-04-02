//go:build unit || !integration

package gpu

import (
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestParsingAMDGPUsWithOne(t *testing.T) {
	output := strings.NewReader(
		`{"card0": {"PCI Bus": "0000:E7:00.0", "VRAM Total Memory (B)": "68702699520", ` +
			`"VRAM Total Used Memory (B)": "10960896", ` +
			`"Card series": "Instinct MI210", "Card model": "0x0c34", ` +
			`"Card vendor": "Advanced Micro Devices, Inc. [AMD/ATI]", "Card SKU":` +
			`"D67301"}}`,
	)

	resources, err := parseRocmSMIOutput(output)
	require.NoError(t, err)
	require.Equal(t, int(resources.GPU), len(resources.GPUs))
	require.Equal(t, float64(0), resources.CPU)
	require.Equal(t, uint64(0), resources.Memory)
	require.Equal(t, uint64(0), resources.Disk)

	gpus := resources.GPUs
	require.Len(t, gpus, 1)
	require.Equal(t, models.GPUVendorAMDATI, gpus[0].Vendor)
	require.Equal(t, uint64(0), gpus[0].Index)
	require.Equal(t, "Instinct MI210", gpus[0].Name)
	require.Equal(t, uint64(65520), gpus[0].Memory)
	require.Equal(t, "0000:e7:00.0", gpus[0].PCIAddress)
}

func TestParsingAMDGPUsWithMany(t *testing.T) {
	output := strings.NewReader(
		`{"card0": {"VRAM Total Memory (B)": "0", "VRAM Total Used ` +
			`Memory (B)": "0", "Card series": "Instinct MI210", "Card ` +
			`model": "0x0c34", "Card vendor": "Advanced Micro Devices, Inc. ` +
			`[AMD/ATI]", "Card SKU": "D67301"}, "card1": {"VRAM Total Memory (B)": ` +
			`"8587837440", "VRAM Total Used Memory (B)": "10960896", "Card ` +
			`series": "Instinct MI210", "Card model": "0x0c34", "Card vendor": ` +
			`"Advanced Micro Devices, Inc. [AMD/ATI]", "Card SKU": "D67301"}, ` +
			`"card2": {"VRAM Total Memory (B)": "17175674880", "VRAM Total Used ` +
			`Memory (B)": "10960896", "Card series": "Instinct MI210", "Card ` +
			`model": "0x0c34", "Card vendor": "Advanced Micro Devices, Inc. ` +
			`[AMD/ATI]", "Card SKU": "D67301"}, "card3": {"VRAM Total Memory (B)": ` +
			`"25763512320", "VRAM Total Used Memory (B)": "10960896", "Card ` +
			`series": "Instinct MI210", "Card model": "0x0c34", "Card vendor": ` +
			`"Advanced Micro Devices, Inc. [AMD/ATI]", "Card SKU": "D67301"} }`,
	)
	resources, err := parseRocmSMIOutput(output)
	require.NoError(t, err)
	require.Equal(t, int(resources.GPU), len(resources.GPUs))
	require.Equal(t, float64(0), resources.CPU)
	require.Equal(t, uint64(0), resources.Memory)
	require.Equal(t, uint64(0), resources.Disk)

	gpus := resources.GPUs
	require.Len(t, gpus, 4)

	for index, gpu := range gpus {
		require.Equal(t, models.GPUVendorAMDATI, gpu.Vendor)
		require.Equal(t, uint64(index), gpu.Index)
		require.Equal(t, "Instinct MI210", gpu.Name)
		require.Equal(t, uint64(index*8190), gpu.Memory)
	}
}

func TestParsingAMDGPUsWithNone(t *testing.T) {
	output := strings.NewReader(`{}`)

	resources, err := parseRocmSMIOutput(output)
	require.NoError(t, err)
	require.Equal(t, int(resources.GPU), len(resources.GPUs))
	require.Equal(t, float64(0), resources.CPU)
	require.Equal(t, uint64(0), resources.Memory)
	require.Equal(t, uint64(0), resources.Disk)

	gpus := resources.GPUs
	require.Len(t, gpus, 0)

}
