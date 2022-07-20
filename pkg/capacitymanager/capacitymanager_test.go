package capacitymanager

import (
	"math"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
)

func TestNewCapacityManager(t *testing.T) {
	m, err := NewCapacityManager(Config{})
	if err != nil {
		t.Fatal(err)
	}

	// Test job defaults are set
	cpuDefault := resourceusage.ConvertCPUString(DefaultJobCPU)
	if math.Abs(cpuDefault-m.resourceRequirementsJobDefault.CPU) > 0.00001 {
		t.Fatalf("default job CPU should be %f, got %f", cpuDefault, m.resourceLimitsTotal.CPU)
	}
	memDefault := resourceusage.ConvertMemoryString(DefaultJobMemory)
	if memDefault != m.resourceRequirementsJobDefault.Memory {
		t.Fatalf("default job memory should be %d, got %d", memDefault, m.resourceLimitsTotal.Memory)
	}
	gpuDefault := resourceusage.ConvertGPUString(DefaultJobGPU)
	if gpuDefault != m.resourceRequirementsJobDefault.GPU {
		t.Fatalf("default job GPU should be %d, got %d", gpuDefault, m.resourceLimitsTotal.GPU)
	}

	// Test job limits cannot be greater than Total limits
	_, err = NewCapacityManager(Config{
		ResourceLimitJob: resourceusage.ResourceUsageConfig{
			CPU: "5",
		},
		ResourceLimitTotal: resourceusage.ResourceUsageConfig{
			CPU: "1",
		},
	})
	if err == nil {
		t.Fatal("job CPU limit should fail when greater than the default total limit (which defaults to the system limit)")
	}
	_, err = NewCapacityManager(Config{
		ResourceLimitJob: resourceusage.ResourceUsageConfig{
			Memory: "5",
		},
		ResourceLimitTotal: resourceusage.ResourceUsageConfig{
			Memory: "1",
		},
	})
	if err == nil {
		t.Fatal("job Memory limit should fail when greater than the default total limit (which defaults to the system limit)")
	}
	_, err = NewCapacityManager(Config{
		ResourceLimitJob: resourceusage.ResourceUsageConfig{
			GPU: "5",
		},
		ResourceLimitTotal: resourceusage.ResourceUsageConfig{
			GPU: "0", // Setting this to 0 makes the `resourceusage.GetSystemResources` call ok
		},
	})
	if err == nil {
		t.Fatal("job GPU limit should fail when greater than the default total limit (which defaults to the system limit)")
	}

	// Test total system limits are set - The parsing is tested in util_test.go
	if m.resourceLimitsTotal.CPU == 0 {
		t.Fatalf("total system CPU should be %f, got %f", 0.0, m.resourceLimitsTotal.CPU)
	}

	// Test that the default job limits are always greater than the job limit set here
	_, err = NewCapacityManager(Config{
		ResourceLimitJob: resourceusage.ResourceUsageConfig{
			GPU: "0",
		},
		ResourceLimitTotal: resourceusage.ResourceUsageConfig{
			GPU: "0", // Setting this to 0 makes the `resourceusage.GetSystemResources` call ok
		},
		ResourceRequirementsDefault: resourceusage.ResourceUsageConfig{
			GPU: "1",
		},
	})
	if err == nil {
		t.Fatal("job GPU limit should fail when less than the default limit")
	}
}
