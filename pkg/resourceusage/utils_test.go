package resourceusage

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/c2h5oh/datasize"
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

// Before all suite
func (suite *ResourceUsageUtilsSuite) SetupAllSuite() {

}

// Before each test
func (suite *ResourceUsageUtilsSuite) SetupTest() {
}

func (suite *ResourceUsageUtilsSuite) TearDownTest() {
}

func (suite *ResourceUsageUtilsSuite) TearDownAllSuite() {

}
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

func (suite *ResourceUsageUtilsSuite) TestParseResourceUsageConfig() {

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
		require.Equal(suite.T(), converted.CPU, test.expected.CPU, "cpu is incorrect")
		require.Equal(suite.T(), converted.Memory, test.expected.Memory, "memory is incorrect")
	}

}

func (suite *ResourceUsageUtilsSuite) TestGetResourceUsageConfig() {

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
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), test.expected.CPU, converted.CPU, "cpu is incorrect")
		require.Equal(suite.T(), test.expected.Memory, converted.Memory, "memory is incorrect")
	}

}

func (suite *ResourceUsageUtilsSuite) TestSystemResources() {

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
			require.Error(suite.T(), err, "an error was expected")
		} else {
			require.NoError(suite.T(), err, "an error was not expected")
			require.Equal(suite.T(), test.expected.CPU, resources.CPU, "cpu is incorrect")
			require.Equal(suite.T(), test.expected.Memory, resources.Memory, "memory is incorrect")
		}

	}

}
