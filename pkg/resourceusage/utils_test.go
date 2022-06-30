package resourceusage

import (
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
)

func c(cpu, mem, disk string) ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    cpu,
		Memory: mem,
		Disk:   disk,
	}
}

func d(cpu float64, mem, disk uint64) ResourceUsageData {
	return ResourceUsageData{
		CPU:    cpu,
		Memory: mem,
		Disk:   disk,
	}
}

func TestConvertResourceUsage(t *testing.T) {

	tests := []struct {
		name     string
		input    ResourceUsageConfig
		expected ResourceUsageData
	}{
		{
			name:     "basic",
			input:    c("500m", "512mb", "1gb"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), (datasize.GB * 1).Bytes()),
		},
		{
			name:     "with i",
			input:    c("500m", "512mi", "1gi"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), (datasize.GB * 1).Bytes()),
		},
		{
			name:     "with spaces",
			input:    c("500 m", "512 mi", "1 gi"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), (datasize.GB * 1).Bytes()),
		},
		{
			name:     "with capitals",
			input:    c("500M", "512MB", "1GI"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), (datasize.GB * 1).Bytes()),
		},
	}

	for _, test := range tests {
		converted, err := ConvertResourceUsageConfig(test.input)
		assert.NoError(t, err)
		assert.Equal(t, converted.CPU, test.expected.CPU, "cpu is incorrect")
		assert.Equal(t, converted.Memory, test.expected.Memory, "memory is incorrect")
		assert.Equal(t, converted.Disk, test.expected.Disk, "disk is incorrect")
	}

}
