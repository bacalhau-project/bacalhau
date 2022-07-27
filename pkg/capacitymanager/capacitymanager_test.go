<<<<<<< HEAD
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
||||||| parent of c1290fd7 (move resourceusage package into capacity manager)
=======
package capacitymanager

import (
	"testing"
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

func TestJobResourceLimits(t *testing.T) {
	runTest := func(
		name string,
		jobResources, jobResourceLimits, defaultJobResourceLimits ResourceUsageConfig,
		expectError bool,
		expectedResult bool,
	) {

		t.Run(name, func(t *testing.T) {
			// manager, err := NewCapacityManager(Config{
			// 	ResourceLimitJob:            jobResourceLimits,
			// 	ResourceRequirementsDefault: defaultJobResourceLimits,
			// })

			// if expectError {
			// 	require.Error(t, err)
			// } else {
			// 	require.NoError(t, err)
			// }
		})

		// job := GetProbeData("")
		// job.Spec.Resources = jobResources

		// result, _, err := computeNode.SelectJob(context.Background(), job)
		// require.NoError(t, err)

		// require.Equal(t, expectedResult, result, fmt.Sprintf("the expcted result was %v, but got %v -- %+v vs %+v", expectedResult, result, jobResources, jobResourceLimits))
	}

	// the job is half the limit
	runTest(
		"Job will run if using half capacity",
		getResources("1", "500Mb", ""),
		getResources("2", "1Gb", ""),
		getResources("100m", "100Mb", ""),
		false,
		true,
	)

	// // the job is on the limit
	// runTest(
	// 	getResources("1", "500Mb", ""),
	// 	getResources("1", "500Mb", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	true,
	// )

	// // the job is over the limit
	// runTest(
	// 	getResources("2", "1Gb", ""),
	// 	getResources("1", "500Mb", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	false,
	// )

	// // test with fractional CPU
	// // the job is less than the limit
	// runTest(
	// 	getResources("250m", "200Mb", ""),
	// 	getResources("1", "500Mb", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	true,
	// )

	// // test when the limit is empty
	// runTest(
	// 	getResources("250m", "200Mb", ""),
	// 	getResources("", "", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	true,
	// )

	// // test when both is empty
	// runTest(
	// 	getResources("", "", ""),
	// 	getResources("", "", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	true,
	// )

	// runTest(
	// 	getResources("", "", ""),
	// 	getResources("250m", "200Mb", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	true,
	// )

	// runTest(
	// 	getResources("300m", "", ""),
	// 	getResources("250m", "200Mb", ""),
	// 	getResources("100m", "100Mb", ""),
	// 	false,
	// )

}
>>>>>>> c1290fd7 (move resourceusage package into capacity manager)
