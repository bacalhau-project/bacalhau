package capacitymanager

import (
	"fmt"
	"os"
	"strings"
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

func TestConstructionErrors(t *testing.T) {
	os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "1")

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

func TestFilter(t *testing.T) {
	os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "1")

	testCases := []struct {
		name           string
		limitTotal     ResourceUsageConfig
		limitJob       ResourceUsageConfig
		defaults       ResourceUsageConfig
		value          ResourceUsageConfig
		expectedOk     bool
		expectedResult ResourceUsageConfig
	}{
		{
			"sanity",
			getResources("10", "10Gb", "10Gb"),
			getResources("2", "2Gb", "2Gb"),
			getResources("", "", ""),
			getResources("1", "1Gb", "1Gb"),
			true,
			getResources("1", "1Gb", "1Gb"),
		},

		// we should get back the default values
		// if we give them no values
		{
			"process defaults",
			getResources("10", "10Gb", "10Gb"),
			getResources("2", "2Gb", "2Gb"),
			getResources("1", "1Gb", "1Gb"),
			getResources("", "", ""),
			true,
			getResources("1", "1Gb", "1Gb"),
		},

		// a job that is over the per job limit
		{
			"over per job limit",
			getResources("10", "10Gb", "10Gb"),
			getResources("2", "2Gb", "2Gb"),
			getResources("", "", ""),
			getResources("4", "4Gb", "4Gb"),
			false,
			getResources("4", "4Gb", "4Gb"),
		},

		// a job that is over the total limit
		{
			"over toal limit",
			getResources("10", "10Gb", "10Gb"),
			getResources("2", "2Gb", "2Gb"),
			getResources("", "", ""),
			getResources("20", "20Gb", "20Gb"),
			false,
			getResources("20", "20Gb", "20Gb"),
		},

		// a job that is over only one limit
		{
			"over per job limit (just CPU)",
			getResources("10", "10Gb", "10Gb"),
			getResources("2", "2Gb", "2Gb"),
			getResources("", "", ""),
			getResources("4", "1Gb", "1Gb"),
			false,
			getResources("4", "1Gb", "1Gb"),
		},

		// job is allowed with mixutre of defaults
		{
			"mixture of defaults - allowed job",
			getResources("10", "10Gb", "10Gb"),
			getResources("2", "2Gb", "2Gb"),
			getResources("", "1Gb", ""),
			getResources("1", "", "500Mb"),
			true,
			getResources("1", "1Gb", "500Mb"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr, err := NewCapacityManager(Config{
				ResourceLimitTotal:          tc.limitTotal,
				ResourceLimitJob:            tc.limitJob,
				ResourceRequirementsDefault: tc.defaults,
			})
			require.NoError(t, err)

			expectedResult := ParseResourceUsageConfig(tc.expectedResult)
			ok, result := mgr.FilterRequirements(ParseResourceUsageConfig(tc.value))
			require.Equal(t, tc.expectedOk, ok)
			require.Equal(t, expectedResult.CPU, result.CPU)
			require.Equal(t, expectedResult.Memory, result.Memory)
			require.Equal(t, expectedResult.Disk, result.Disk)
		})
	}

}

func TestGetNextItems(t *testing.T) {
	os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "1")

	// this means we can test "long lived" jobs that use resources
	// for longer than other jobs
	type TestJob struct {
		iterations int
		usage      ResourceUsageConfig
	}

	testCases := []struct {
		name  string
		limit ResourceUsageConfig
		jobs  []TestJob
		// a csv array of the currently running jobs for each iteration
		// this will be based on the "iterations" setting of the job
		// as well as the capacity manager's current state in terms of scheduling
		expectedLogs []string
	}{
		{
			"sanity",
			getResources("10", "10Gb", "10Gb"),
			[]TestJob{
				{
					1,
					getResources("2", "2Gb", "2Gb"),
				},
			},
			// a single job on it's own once
			[]string{
				"0",
				"",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr, err := NewCapacityManager(Config{
				ResourceLimitTotal:          tc.limit,
				ResourceLimitJob:            tc.limit,
				ResourceRequirementsDefault: getResources("", "", ""),
			})
			require.NoError(t, err)

			iterationMap := map[string]int{}
			counterMap := map[string]int{}
			logs := []string{}

			for id, job := range tc.jobs {
				idString := fmt.Sprintf("%d", id)
				iterationMap[idString] = job.iterations
				mgr.AddToBacklog(idString, ParseResourceUsageConfig(job.usage))
			}

			for {

				// get the items we have space to run
				nextItems := mgr.GetNextItems()

				// mark each new item as active and start it's
				// iteration counter at zero
				for _, id := range nextItems {
					mgr.MoveToActive(id)
					counterMap[id] = 0
				}

				running := []string{}
				toRemove := []string{}
				// loop over currently active items and increment
				// the iteration counter and remove them
				// if they have "completed"
				mgr.active.Iterate(func(item CapacityManagerItem) {
					counterMap[item.ID]++
					if counterMap[item.ID] > iterationMap[item.ID] {
						toRemove = append(toRemove, item.ID)
					} else {
						running = append(running, item.ID)
					}
				})

				logs = append(logs, strings.Join(running, ","))

				for _, id := range toRemove {
					mgr.Remove(id)
				}

				// this means we've cleared out all the jobs
				if mgr.backlog.Count() <= 0 && mgr.active.Count() <= 0 {
					break
				}
			}

			require.Equal(t, strings.Join(tc.expectedLogs, "\n"), strings.Join(logs, "\n"))
		})
	}

}

// func TestManagerSelection(t *testing.T) {

// 	testCases := []struct {
// 		name         string
// 		limitTotal   ResourceUsageConfig
// 		limitJob     ResourceUsageConfig
// 		jobDefaults  ResourceUsageConfig
// 		usedCapacity ResourceUsageConfig
// 		jobSpec      ResourceUsageConfig

// 		expectedResult bool
// 	}{
// 		{
// 			// 10 in total
// 			// 2 per job limit
// 			// defaults blank
// 			// 1 being used
// 			// 1 is the actual job
// 			// should select
// 			"sanity",
// 			getResources("10", "10Gb", "10Gb"),
// 			getResources("2", "2Gb", "2Gb"),
// 			getResources("", "", ""),
// 			getResources("1", "1Gb", "1Gb"),
// 			getResources("1", "1Gb", "1Gb"),
// 			true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			_, err := NewCapacityManager(Config{
// 				ResourceLimitTotal:          tc.limitTotal,
// 				ResourceLimitJob:            tc.limitJob,
// 				ResourceRequirementsDefault: tc.defaults,
// 			})
// 			if tc.expectError != "" {
// 				require.Error(t, err)
// 				require.Equal(t, tc.expectError, err.Error())
// 			} else {
// 				require.NoError(t, err)
// 			}
// 		})
// 	}

// }

// the job is half the limit
// runTest(
// 	getResources("1", "500Mb", ""),
// 	getResources("2", "1Gb", ""),
// 	getResources("100m", "100Mb", ""),
// 	true,
// )

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

// func TestManagerConstruction(t *testing.T) {
// 	runTest := func(
// 		name string,
// 		jobResourceLimits, defaultJobResourceLimits ResourceUsageConfig,
// 		expectError bool,
// 	) {

// 		t.Run(name, func(t *testing.T) {
// 			_, err := NewCapacityManager(Config{
// 				ResourceLimitJob:            jobResourceLimits,
// 				ResourceRequirementsDefault: defaultJobResourceLimits,
// 			})
// 			if expectError {
// 				require.Error(t, err)
// 			} else {
// 				require.NoError(t, err)
// 			}
// 		})

// 		// job := GetProbeData("")
// 		// job.Spec.Resources = jobResources

// 		// result, _, err := computeNode.SelectJob(context.Background(), job)
// 		// require.NoError(t, err)

// 		// require.Equal(t, expectedResult, result, fmt.Sprintf("the expcted result was %v, but got %v -- %+v vs %+v", expectedResult, result, jobResources, jobResourceLimits))
// 	}

// }
