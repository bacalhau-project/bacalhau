package resourceusage

import (
	"fmt"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
)

func c(cpu, mem string) ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    cpu,
		Memory: mem,
	}
}

func d(cpu float64, mem uint64) ResourceUsageData {
	return ResourceUsageData{
		CPU:    cpu,
		Memory: mem,
	}
}

func TestParseResourceUsageConfig(t *testing.T) {

	tests := []struct {
		name     string
		input    ResourceUsageConfig
		expected ResourceUsageData
	}{
		{
			name:     "basic",
			input:    c("500m", "512mb"),
			expected: d(0.5, (datasize.MB * 512).Bytes()),
		},
		{
			name:     "with i",
			input:    c("500m", "512mi"),
			expected: d(0.5, (datasize.MB * 512).Bytes()),
		},
		{
			name:     "with spaces",
			input:    c("500 m", "512 mi"),
			expected: d(0.5, (datasize.MB * 512).Bytes()),
		},
		{
			name:     "with capitals",
			input:    c("500M", "512MB"),
			expected: d(0.5, (datasize.MB * 512).Bytes()),
		},
		{
			name:     "empty",
			input:    c("", ""),
			expected: d(0, (datasize.B * 0).Bytes()),
		},
	}

	for _, test := range tests {
		converted, err := ParseResourceUsageConfig(test.input)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		assert.NoError(t, err)
		assert.Equal(t, converted.CPU, test.expected.CPU, "cpu is incorrect")
		assert.Equal(t, converted.Memory, test.expected.Memory, "memory is incorrect")
	}

}

func TestGetResourceUsageConfig(t *testing.T) {

	tests := []struct {
		name     string
		input    ResourceUsageData
		expected ResourceUsageConfig
	}{
		{
			name:     "basic",
			input:    d(0.5, (datasize.MB * 512).Bytes()),
			expected: c("500m", "512MB"),
		},
	}

	for _, test := range tests {
		converted, err := GetResourceUsageConfig(test.input)
		assert.NoError(t, err)
		assert.Equal(t, test.expected.CPU, converted.CPU, "cpu is incorrect")
		assert.Equal(t, test.expected.Memory, converted.Memory, "memory is incorrect")
	}

}
