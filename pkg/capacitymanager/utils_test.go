//go:build !integration

package capacitymanager

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/pbnjay/memory"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ResourceUsageUtilsSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestResourceUsageUtilsSuite(t *testing.T) {
	suite.Run(t, new(ResourceUsageUtilsSuite))
}

// Before each test
func (suite *ResourceUsageUtilsSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
}

func c(cpu, mem, gpu string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    cpu,
		Memory: mem,
		GPU:    gpu,
	}
}

func d(cpu float64, mem uint64, gpu uint64) model.ResourceUsageData {
	return model.ResourceUsageData{
		CPU:    cpu,
		Memory: mem,
		GPU:    gpu,
	}
}

func (suite *ResourceUsageUtilsSuite) TestParseResourceUsageConfig() {

	tests := []struct {
		name     string
		input    model.ResourceUsageConfig
		expected model.ResourceUsageData
	}{
		{
			name:     "basic",
			input:    c("500m", "512mb", "2"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), 2),
		},
		{
			name:     "invalid GPU 1",
			input:    c("500m", "512mb", "-2"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), 0),
		},
		{
			name:     "invalid GPU 2",
			input:    c("500m", "512mb", "1.1"),
			expected: d(0.5, (datasize.MB * 512).Bytes(), 0),
		},
		{
			name:     "with i",
			input:    c("500m", "512mi", ""),
			expected: d(0.5, (datasize.MB * 512).Bytes(), 0),
		},
		{
			name:     "with spaces",
			input:    c("500 m", "512 mi", " "),
			expected: d(0.5, (datasize.MB * 512).Bytes(), 0),
		},
		{
			name:     "with capitals",
			input:    c("500M", "512MB", ""),
			expected: d(0.5, (datasize.MB * 512).Bytes(), 0),
		},
		{
			name:     "empty",
			input:    c("", "", ""),
			expected: d(0, (datasize.B * 0).Bytes(), 0),
		},
	}

	for _, test := range tests {

		suite.Run(test.name, func() {
			converted := ParseResourceUsageConfig(test.input)
			require.Equal(suite.T(), converted.CPU, test.expected.CPU, "cpu is incorrect")
			require.Equal(suite.T(), converted.Memory, test.expected.Memory, "memory is incorrect")
			require.Equal(suite.T(), converted.GPU, test.expected.GPU, "gpu is incorrect")
		})

	}

}

func (suite *ResourceUsageUtilsSuite) TestSystemResources() {

	tests := []struct {
		name        string
		shouldError bool
		input       model.ResourceUsageConfig
		expected    model.ResourceUsageData
	}{
		{
			name:        "should return 80% of what the system has",
			shouldError: false,
			input:       c("", "", ""),
			expected:    d(float64(runtime.NumCPU())*0.8, memory.TotalMemory()*80/100, numSystemGPUsNoError()),
		},
		{
			name:        "should return the configured CPU amount",
			shouldError: false,
			input:       c("100m", "", ""),
			expected:    d(float64(0.1), memory.TotalMemory()*80/100, numSystemGPUsNoError()),
		},
		{
			name:        "should return the configured Memory amount",
			shouldError: false,
			input:       c("", "100Mb", ""),
			expected:    d(float64(runtime.NumCPU())*0.8, ConvertMemoryString("100Mb"), numSystemGPUsNoError()),
		},
		{
			name:        "should error with too many CPUs asked for",
			shouldError: true,
			input:       c(fmt.Sprintf("%f", float64(runtime.NumCPU())*2), "", ""),
		},
		{
			name:        "should error with too much Memory asked for",
			shouldError: true,
			input:       c("", fmt.Sprintf("%db", memory.TotalMemory()*2), ""),
		},
		{
			name:        "should error with too much GPU asked for",
			shouldError: true,
			input:       c("", "", "5"),
		},
	}

	for _, test := range tests {

		suite.Run(test.name, func() {
			resources, err := getSystemResources(test.input)

			if test.shouldError {
				require.Error(suite.T(), err, "an error was expected")
			} else {
				require.NoError(suite.T(), err, "an error was not expected")
				require.Equal(suite.T(), test.expected.CPU, resources.CPU, "cpu is incorrect")
				require.Equal(suite.T(), test.expected.Memory, resources.Memory, "memory is incorrect")
				require.Equal(suite.T(), test.expected.GPU, resources.GPU, "GPU is incorrect")
			}
		})

	}
}

func TestSubtractResourceUsage(t *testing.T) {
	res := subtractResourceUsage(
		model.ResourceUsageData{
			CPU:    0.5,
			Memory: (datasize.MB * 512).Bytes(),
			GPU:    2,
		},
		model.ResourceUsageData{
			CPU:    1,
			Memory: (datasize.GB * 1).Bytes(),
			GPU:    4,
		},
	)
	if res.CPU != 0.5 {
		t.Errorf("CPU was incorrect: %f", res.CPU)
	}
	if res.Memory != (datasize.MB * 512).Bytes() {
		t.Errorf("Memory was incorrect: %d", res.Memory)
	}
	if res.GPU != 2 {
		t.Errorf("GPU was incorrect: %d", res.GPU)
	}
}

func TestCheckResourceUsage(t *testing.T) {
	// Test when resources are ok, should return true
	ok := checkResourceUsage(
		model.ResourceUsageData{
			CPU:    0.5,
			Memory: (datasize.MB * 512).Bytes(),
			GPU:    2,
		},
		model.ResourceUsageData{
			CPU:    1,
			Memory: (datasize.GB * 1).Bytes(),
			GPU:    4,
		},
	)
	if !ok {
		t.Error("checkResourceUsage returned false")
	}

	// test when resources are not ok
	ok = checkResourceUsage(
		model.ResourceUsageData{
			CPU:    0.5,
			Memory: (datasize.MB * 512).Bytes(),
			GPU:    2,
		},
		model.ResourceUsageData{
			CPU:    1,
			Memory: (datasize.GB * 1).Bytes(),
			GPU:    0,
		},
	)
	if ok {
		t.Error("checkResourceUsage returned true")
	}
	ok = checkResourceUsage(
		model.ResourceUsageData{
			CPU:    0.5,
			Memory: (datasize.MB * 512).Bytes(),
			GPU:    2,
		},
		model.ResourceUsageData{
			CPU:    0,
			Memory: (datasize.GB * 1).Bytes(),
			GPU:    4,
		},
	)
	if ok {
		t.Error("checkResourceUsage returned true")
	}
	ok = checkResourceUsage(
		model.ResourceUsageData{
			CPU:    0.5,
			Memory: (datasize.MB * 512).Bytes(),
			GPU:    2,
		},
		model.ResourceUsageData{
			CPU:    1,
			Memory: (datasize.GB * 0).Bytes(),
			GPU:    4,
		},
	)
	if ok {
		t.Error("checkResourceUsage returned true")
	}
}

func numSystemGPUsNoError() uint64 {
	numGPUs, err := numSystemGPUs()
	if err != nil {
		return 0
	}
	return numGPUs
}
