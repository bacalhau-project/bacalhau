//go:build !integration

package capacitymanager

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

type MockCapacityTracker struct {
	backlog []CapacityManagerItem
	active  []CapacityManagerItem
}

func (m *MockCapacityTracker) addToBacklog(item CapacityManagerItem) {
	m.backlog = append(m.backlog, item)
}

func (m *MockCapacityTracker) addToActive(item CapacityManagerItem) {
	m.active = append(m.active, item)
}

func (m *MockCapacityTracker) moveToActive(itemID string) {
	for i, v := range m.backlog {
		if v.Shard.Job.ID == itemID {
			m.backlog = append(m.backlog[:i], m.backlog[i+1:]...)
			m.addToActive(v)
			return
		}
	}
}

func (m *MockCapacityTracker) remove(itemID string) {
	for i, v := range m.backlog {
		if v.Shard.Job.ID == itemID {
			m.backlog = append(m.backlog[:i], m.backlog[i+1:]...)
			break
		}
	}
	for i, v := range m.active {
		if v.Shard.Job.ID == itemID {
			m.active = append(m.active[:i], m.active[i+1:]...)
			break
		}
	}
}

func (m *MockCapacityTracker) BacklogIterator(handler func(item CapacityManagerItem)) {
	for _, item := range m.backlog {
		handler(item)
	}
}

func (m *MockCapacityTracker) ActiveIterator(handler func(item CapacityManagerItem)) {
	for _, item := range m.active {
		handler(item)
	}
}

func getResources(c, m, d string) model.ResourceUsageConfig {
	return model.ResourceUsageConfig{
		CPU:    c,
		Memory: m,
		Disk:   d,
	}
}

//nolint:unused
func getResourcesArray(data [][]string) []model.ResourceUsageConfig {
	var res []model.ResourceUsageConfig
	for _, d := range data {
		res = append(res, getResources(d[0], d[1], d[2]))
	}
	return res
}

func TestConstructionErrors(t *testing.T) {
	os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "1")
	defer os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "")

	capacityTracker := &MockCapacityTracker{}

	testCases := []struct {
		name        string
		limitTotal  model.ResourceUsageConfig
		limitJob    model.ResourceUsageConfig
		defaults    model.ResourceUsageConfig
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
			_, err := NewCapacityManager(capacityTracker, Config{
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
	defer os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "")

	capacityTracker := &MockCapacityTracker{}

	testCases := []struct {
		name           string
		limitTotal     model.ResourceUsageConfig
		limitJob       model.ResourceUsageConfig
		defaults       model.ResourceUsageConfig
		value          model.ResourceUsageConfig
		expectedOk     bool
		expectedResult model.ResourceUsageConfig
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
			mgr, err := NewCapacityManager(capacityTracker, Config{
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
	defer os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", "")

	capacityTracker := &MockCapacityTracker{}

	// this means we can test "long lived" jobs that use resources
	// for longer than other jobs
	type TestJob struct {
		iterations int
		usage      model.ResourceUsageConfig
	}

	testCases := []struct {
		name  string
		limit model.ResourceUsageConfig
		jobs  []TestJob
		// a csv array of the currently running jobs for each iteration
		// this will be based on the "iterations" setting of the job
		// as well as the capacity manager's current state in terms of scheduling
		expectedLogs []string
	}{

		// simple one off job where there is more than enough space to run it
		// and it only lasts for one iteration
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

		// a sequence of equally sized and lasting jobs
		// this should end up with 2 phases of 2 jobs each
		{
			"equal jobs",
			getResources("10", "10Gb", "10Gb"),
			[]TestJob{
				{1, getResources("5", "5Gb", "5Gb")},
				{1, getResources("5", "5Gb", "5Gb")},
				{1, getResources("5", "5Gb", "5Gb")},
				{1, getResources("5", "5Gb", "5Gb")},
			},
			// first two jobs then second two jobs
			[]string{
				"0,1",
				"2,3",
				"",
			},
		},

		// one long running large job and lots
		// of smaller jobs scheduled around it
		{
			"one large job",
			getResources("10", "10Gb", "10Gb"),
			[]TestJob{
				{3, getResources("9", "9Gb", "9Gb")},
				{1, getResources("1", "1Gb", "1Gb")},
				{1, getResources("1", "1Gb", "1Gb")},
				{1, getResources("1", "1Gb", "1Gb")},
			},
			// first two jobs then second two jobs
			[]string{
				"0,1",
				"0,2",
				"0,3",
				"",
			},
		},

		// there is not space for a big job
		// until some others have finished
		{
			"big job waits",
			getResources("10", "10Gb", "10Gb"),
			[]TestJob{
				{1, getResources("4", "4Gb", "4Gb")},
				{1, getResources("4", "4Gb", "4Gb")},
				{1, getResources("10", "10Gb", "10Gb")},
			},
			// first two jobs then second two jobs
			[]string{
				"0,1",
				"2",
				"",
			},
		},

		// things are scheduled that were added
		// later than earlier things until
		// there is space to run the earlier thing
		{
			"schedule ahead",
			getResources("10", "10Gb", "10Gb"),
			[]TestJob{
				{3, getResources("8", "8Gb", "8Gb")},
				{1, getResources("4", "4Gb", "4Gb")},
				{1, getResources("2", "2Gb", "2Gb")},
				{1, getResources("2", "2Gb", "2Gb")},
			},
			// first two jobs then second two jobs
			[]string{
				"0,2",
				"0,3",
				"0",
				"1",
				"",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr, err := NewCapacityManager(capacityTracker, Config{
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
				counterMap[idString] = 0
				iterationMap[idString] = job.iterations
				shard := model.JobShard{
					Job:   &model.Job{ID: idString},
					Index: 0,
				}
				capacityTracker.addToBacklog(CapacityManagerItem{
					Shard:        shard,
					Requirements: ParseResourceUsageConfig(job.usage),
				})
			}

			for {

				toRemove := []string{}
				running := []string{}

				// loop over currently active items and increment
				// the iteration counter and remove them
				// if they have "completed"
				capacityTracker.ActiveIterator(func(item CapacityManagerItem) {
					counterMap[item.Shard.Job.ID]++
					if counterMap[item.Shard.Job.ID] >= iterationMap[item.Shard.Job.ID] {
						toRemove = append(toRemove, item.Shard.Job.ID)
					} else {
						running = append(running, item.Shard.Job.ID)
					}
				})

				for _, id := range toRemove {
					capacityTracker.remove(id)
				}

				// get the items we have space to run
				nextItems := mgr.GetNextItems()

				// mark each new item as active and start it's
				// iteration counter at zero
				for _, shard := range nextItems {
					capacityTracker.moveToActive(shard.Job.ID)
					running = append(running, shard.Job.ID)
				}

				sort.Strings(running)
				logs = append(logs, strings.Join(running, ","))

				// this means we've cleared out all the jobs
				if len(capacityTracker.backlog) <= 0 && len(capacityTracker.active) <= 0 {
					break
				}
			}

			require.Equal(t, strings.Join(tc.expectedLogs, "\n"), strings.Join(logs, "\n"))
		})
	}
}

func TestNewCapacityManager(t *testing.T) {
	capacityTracker := &MockCapacityTracker{}

	m, err := NewCapacityManager(capacityTracker, Config{})
	if err != nil {
		t.Fatal(err)
	}

	// Test job defaults are set
	cpuDefault := ConvertCPUString(DefaultJobCPU)
	if math.Abs(cpuDefault-m.resourceRequirementsJobDefault.CPU) > 0.00001 {
		t.Fatalf("default job CPU should be %f, got %f", cpuDefault, m.resourceLimitsTotal.CPU)
	}
	memDefault := ConvertMemoryString(DefaultJobMemory)
	if memDefault != m.resourceRequirementsJobDefault.Memory {
		t.Fatalf("default job memory should be %d, got %d", memDefault, m.resourceLimitsTotal.Memory)
	}
	gpuDefault := ConvertGPUString(DefaultJobGPU)
	if gpuDefault != m.resourceRequirementsJobDefault.GPU {
		t.Fatalf("default job GPU should be %d, got %d", gpuDefault, m.resourceLimitsTotal.GPU)
	}

	// Test job limits cannot be greater than Total limits
	_, err = NewCapacityManager(capacityTracker, Config{
		ResourceLimitJob: model.ResourceUsageConfig{
			CPU: "5",
		},
		ResourceLimitTotal: model.ResourceUsageConfig{
			CPU: "1",
		},
	})
	if err == nil {
		t.Fatal("job CPU limit should fail when greater than the default total limit (which defaults to the system limit)")
	}
	_, err = NewCapacityManager(capacityTracker, Config{
		ResourceLimitJob: model.ResourceUsageConfig{
			Memory: "5",
		},
		ResourceLimitTotal: model.ResourceUsageConfig{
			Memory: "1",
		},
	})
	if err == nil {
		t.Fatal("job Memory limit should fail when greater than the default total limit (which defaults to the system limit)")
	}
	_, err = NewCapacityManager(capacityTracker, Config{
		ResourceLimitJob: model.ResourceUsageConfig{
			GPU: "5",
		},
		ResourceLimitTotal: model.ResourceUsageConfig{
			GPU: "0", // Setting this to 0 makes the `GetSystemResources` call ok
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

	gpus, _ := numSystemGPUs()
	if gpus == 0 {
		// This test only works when the CI runner has no GPUs. Don't run it if
		// we have GPUs. TODO: figure out why.
		_, err = NewCapacityManager(capacityTracker, Config{
			ResourceLimitJob: model.ResourceUsageConfig{
				GPU: "0",
			},
			ResourceLimitTotal: model.ResourceUsageConfig{
				GPU: "0", // Setting this to 0 makes the `GetSystemResources` call ok
			},
			ResourceRequirementsDefault: model.ResourceUsageConfig{
				GPU: "1",
			},
		})
		if err == nil {
			t.Fatal("job GPU limit should fail when less than the default limit")
		}
	}
}

func TestFilterRequirements(t *testing.T) {
	capacityTracker := &MockCapacityTracker{}

	m, err := NewCapacityManager(capacityTracker, Config{
		ResourceLimitTotal: model.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0", // TODO:  Can't test GPUs because we can't mock
		},
		ResourceRequirementsDefault: model.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ok, req := m.FilterRequirements(
		model.ResourceUsageData{},
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
	capacityTracker := &MockCapacityTracker{}

	m, err := NewCapacityManager(capacityTracker, Config{
		ResourceLimitTotal: model.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0", // TODO:  Can't test GPUs because we can't mock
		},
		ResourceRequirementsDefault: model.ResourceUsageConfig{
			CPU:    "1",
			Memory: "1Gi",
			GPU:    "0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	capacityTracker.addToActive(CapacityManagerItem{
		Shard: model.JobShard{
			Job:   &model.Job{ID: "test"},
			Index: 0,
		},
		Requirements: model.ResourceUsageData{
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
	gpus, _ := numSystemGPUs()
	if res.GPU != gpus {
		t.Errorf("Should be using all GPU, but got %d", res.GPU)
	}
}
