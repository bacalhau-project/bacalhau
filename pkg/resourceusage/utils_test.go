package resourceusage

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/pbnjay/memory"
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
		converted := ParseResourceUsageConfig(test.input)
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

func TestSystemResources(t *testing.T) {

	tests := []struct {
		name        string
		shouldError bool
		input       ResourceUsageConfig
		expected    ResourceUsageData
	}{
		{
			name:        "should return what the system has",
			shouldError: false,
			input:       c("", ""),
			expected:    d(float64(runtime.NumCPU()), memory.TotalMemory()),
		},
		{
			name:        "should return the configured CPU amount",
			shouldError: false,
			input:       c("100m", ""),
			expected:    d(float64(0.1), memory.TotalMemory()),
		},
		{
			name:        "should return the configured Memory amount",
			shouldError: false,
			input:       c("", "100Mb"),
			expected:    d(float64(runtime.NumCPU()), ConvertMemoryString("100Mb")),
		},
		{
			name:        "should error with too many CPUs asked for",
			shouldError: true,
			input:       c(fmt.Sprintf("%f", float64(runtime.NumCPU())*2), ""),
		},
		{
			name:        "should error with too much Memory asked for",
			shouldError: true,
			input:       c("", fmt.Sprintf("%db", memory.TotalMemory()*2)),
		},
	}

	for _, test := range tests {
		resources, err := GetSystemResources(test.input)

		if test.shouldError {
			assert.Error(t, err, "an error was expected")
		} else {
			assert.NoError(t, err, "an error was not expected")
			assert.Equal(t, test.expected.CPU, resources.CPU, "cpu is incorrect")
			assert.Equal(t, test.expected.Memory, resources.Memory, "memory is incorrect")
		}

	}

}
