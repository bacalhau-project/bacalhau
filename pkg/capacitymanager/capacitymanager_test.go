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
