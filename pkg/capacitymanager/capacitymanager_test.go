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

func TestFilterRequirements(t *testing.T) {
	m, err := NewCapacityManager(Config{
		ResourceLimitTotal: resourceusage.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0", // TODO:  Can't test GPUs because we can't mock
		},
		ResourceRequirementsDefault: resourceusage.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ok, req := m.FilterRequirements(
		resourceusage.ResourceUsageData{},
	)
	if !ok {
		t.Error("Should be ok, but is not")
	}
	if req.CPU != 1 {
		t.Errorf("CPU should be 1, but got %f", req.CPU)
	}
	if req.Memory != 1073741824 {
		t.Errorf("Memory should be 1073741824, but got %d", req.Memory)
	}
	if req.GPU != 0 {
		t.Errorf("GPU should be 0, but got %d", req.GPU)
	}
}

func TestGetFreeSpace(t *testing.T) {
	m, err := NewCapacityManager(Config{
		ResourceLimitTotal: resourceusage.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0", // TODO:  Can't test GPUs because we can't mock
		},
		ResourceRequirementsDefault: resourceusage.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.active.Add(CapacityManagerItem{
		ID: "test",
		Requirements: resourceusage.ResourceUsageData{
			CPU:    1,
			Memory: 1073741824,
			GPU:    0,
		},
	})
	res := m.GetFreeSpace()
	if res.CPU != 0 {
		t.Errorf("Should be using all CPU, but got %f", res.CPU)
	}
	if res.Memory != 0 {
		t.Errorf("Should be using all Memory, but got %d", res.Memory)
	}
	if res.GPU != 0 {
		t.Errorf("Should be using all GPU, but got %d", res.GPU)
	}
}
