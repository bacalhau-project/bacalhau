package capacitymanager

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func getResources(c, m, d string) ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused,deadcode
func getResourcesArray(data [][]string) []ResourceUsageConfig {
	var res []ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

func TestManagerConstructionErrors(t *testing.T) {

	testCases := []struct {
		name        string
		limitTotal  ResourceUsageConfig
		limitJob    ResourceUsageConfig
		defaults    ResourceUsageConfig
		expectError string
	}{
		{
			"sanity",
			getResources("1", "1Gb", "1Gb"),
			getResources("0.1", "100Mb", "100Mb"),
			getResources("", "", ""),
			"",
		},
		{
			"job CPU > total",
			getResources("1", "1Gb", "1Gb"),
			getResources("2", "100Mb", "100Mb"),
			getResources("", "", ""),
			fmt.Sprintf("job resource limit CPU %f is greater than total system limit %f", float64(2), float64(1)),
		},
		{
			"job Memory > total",
			getResources("1", "1Gb", "1Gb"),
			getResources("0.1", "2Gb", "100Mb"),
			getResources("", "", ""),
			fmt.Sprintf("job resource limit memory %d is greater than total system limit %d", ConvertMemoryString("2Gb"), ConvertMemoryString("1Gb")),
		},
		{
			"job Disk > total",
			getResources("1", "1Gb", "1Gb"),
			getResources("0.1", "100Mb", "2Gb"),
			getResources("", "", ""),
			fmt.Sprintf("job resource limit disk %d is greater than total system limit %d", ConvertMemoryString("2Gb"), ConvertMemoryString("1Gb")),
		},
		{
			"default CPU > job",
			getResources("1", "1Gb", "1Gb"),
			getResources("0.1", "100Mb", "100Mb"),
			getResources("0.2", "", ""),
			fmt.Sprintf("default job resource CPU %f is greater than limit %f", float64(0.2), float64(0.1)),
		},
		{
			"default Memory > job",
			getResources("1", "1Gb", "1Gb"),
			getResources("0.1", "100Mb", "100Mb"),
			getResources("", "200Mb", ""),
			fmt.Sprintf("default job resource memory %d is greater than limit %d", ConvertMemoryString("200Mb"), ConvertMemoryString("100Mb")),
		},
		{
			"default CPU > job",
			getResources("1", "1Gb", "1Gb"),
			getResources("0.1", "100Mb", "100Mb"),
			getResources("", "", "200Mb"),
			fmt.Sprintf("default job resource disk %d is greater than limit %d", ConvertMemoryString("200Mb"), ConvertMemoryString("100Mb")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCapacityManager(Config{
				ResourceLimitTotal:          tc.limitTotal,
				ResourceLimitJob:            tc.limitJob,
				ResourceRequirementsDefault: tc.defaults,
			})
			if tc.expectError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expectError, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}

}
