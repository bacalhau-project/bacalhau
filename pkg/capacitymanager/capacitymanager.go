package capacitymanager

import (
	"fmt"
)

const DefaultJobCPU = "100m"
const DefaultJobMemory = "100Mb"
const DefaultJobGPU = "0"

// configures our maximum allowance for all items,
// single item and defaults for single item
type Config struct {
	// the total amount of CPU and RAM we want to
	// give to running bacalhau jobs
	ResourceLimitTotal ResourceUsageConfig
	// limit the max CPU / Memory usage for any single job
	ResourceLimitJob ResourceUsageConfig
	// if a job does not state how much CPU or Memory is used
	// what values should we assume?
	ResourceRequirementsDefault ResourceUsageConfig
}

type CapacityManagerItem struct {
	ID           string
	Requirements ResourceUsageData
}

type CapacityTracker interface {
	// A map of jobs the compute node has decided to bid on according to
	// the JobSelectionPolicy, but which have not yet been accepted by the
	// requester node that initated the job.
	BacklogIterator(handler func(item CapacityManagerItem))

	// jobs we are currently bidding on
	// this is "potential" usage because accepted bids
	// will start coming in (which turns a BiddingJob into a RunningJob)
	// so when we ask "how much capacity are we using"
	// we need to sum "RunningJobs" and a coeffcieint of "BiddingJobs"
	// the coefficient represents how much we over promise our capacity
	// based on bids not being accepted
	ActiveIterator(handler func(item CapacityManagerItem))
}

type CapacityManager struct {
	// The configuration used to create this compute node.
	config Config //nolint:gocritic

	// both of these are is either what the physical CPU / memory values are
	// or the user defined limits from the config
	// if the user defined limits are more than the actual physical
	// amounts we will get an error
	// if job resource limit is more than total resource limit
	// then we will error (in the case both values are supplied)
	resourceLimitsTotal            ResourceUsageData
	resourceLimitsJob              ResourceUsageData
	resourceRequirementsJobDefault ResourceUsageData

	capacityTracker CapacityTracker
}

func NewCapacityManager( //nolint:funlen,gocyclo
	capacityTracker CapacityTracker,
	config Config, //nolint:gocritic
) (*CapacityManager, error) {
	// assign the default config values
	useConfig := config

	// if we've not been given a default job resource limit
	// then let's use some sensible defaults (which are low on purpose)
	if useConfig.ResourceRequirementsDefault.CPU == "" {
		useConfig.ResourceRequirementsDefault.CPU = DefaultJobCPU
	}

	if useConfig.ResourceRequirementsDefault.Memory == "" {
		useConfig.ResourceRequirementsDefault.Memory = DefaultJobMemory
	}

	if useConfig.ResourceRequirementsDefault.GPU == "" {
		useConfig.ResourceRequirementsDefault.GPU = DefaultJobGPU
	}

	resourceLimitsTotal, err := getSystemResources(useConfig.ResourceLimitTotal)
	if err != nil {
		return nil, err
	}

	// this is the per job resource limit - i.e. no job can use more than this
	// if no values are given - then we will use the system available resources
	resourceLimitsJob := ParseResourceUsageConfig(useConfig.ResourceLimitJob)

	// the default value for how much CPU / RAM one job says it needs
	// this is for when a job is submitted with no values for CPU & RAM
	// we will assign these values to it
	resourceRequirementsJobDefault := ParseResourceUsageConfig(useConfig.ResourceRequirementsDefault)

	// if we don't have a limit on job size
	// then let's use the total resources we have on the system
	if resourceLimitsJob.CPU <= 0 {
		resourceLimitsJob.CPU = resourceLimitsTotal.CPU
	}

	if resourceLimitsJob.Memory <= 0 {
		resourceLimitsJob.Memory = resourceLimitsTotal.Memory
	}

	if resourceLimitsJob.Disk <= 0 {
		resourceLimitsJob.Disk = resourceLimitsTotal.Disk
	}

	if resourceLimitsJob.GPU <= 0 {
		resourceLimitsJob.GPU = resourceLimitsTotal.GPU
	}

	// we can't have one job that uses more than we have
	if resourceLimitsJob.CPU > resourceLimitsTotal.CPU {
		return nil, fmt.Errorf("job resource limit CPU %f is greater than total system limit %f",
			resourceLimitsJob.CPU, resourceLimitsTotal.CPU,
		)
	}

	if resourceLimitsJob.Memory > resourceLimitsTotal.Memory {
		return nil, fmt.Errorf(
			"job resource limit memory %d is greater than total system limit %d",
			resourceLimitsJob.Memory, resourceLimitsTotal.Memory,
		)
	}

	if resourceLimitsJob.Disk > resourceLimitsTotal.Disk {
		return nil, fmt.Errorf(
			"job resource limit disk %d is greater than total system limit %d",
			resourceLimitsJob.Disk, resourceLimitsTotal.Disk,
		)
	}

	if resourceLimitsJob.GPU > resourceLimitsTotal.GPU {
		return nil, fmt.Errorf(
			"job resource limit GPU %d is greater than total system limit %d",
			resourceLimitsJob.GPU, resourceLimitsTotal.GPU,
		)
	}

	// the default for job requirements can't be more than our job limit
	// or we'll never accept any jobs and so this is classed as a config error
	if resourceRequirementsJobDefault.CPU > resourceLimitsJob.CPU {
		return nil, fmt.Errorf(
			"default job resource CPU %f is greater than limit %f",
			resourceRequirementsJobDefault.CPU, resourceLimitsJob.CPU,
		)
	}

	if resourceRequirementsJobDefault.Memory > resourceLimitsJob.Memory {
		return nil, fmt.Errorf(
			"default job resource memory %d is greater than limit %d",
			resourceRequirementsJobDefault.Memory, resourceLimitsJob.Memory,
		)
	}

	if resourceRequirementsJobDefault.Disk > resourceLimitsJob.Disk {
		return nil, fmt.Errorf(
			"default job resource disk %d is greater than limit %d",
			resourceRequirementsJobDefault.Disk, resourceLimitsJob.Disk,
		)
	}

	if resourceRequirementsJobDefault.GPU > resourceLimitsJob.GPU {
		return nil, fmt.Errorf(
			"default job resource GPU %d is greater than limit %d",
			resourceRequirementsJobDefault.GPU, resourceLimitsJob.GPU,
		)
	}

	return &CapacityManager{
		config:                         useConfig,
		capacityTracker:                capacityTracker,
		resourceLimitsTotal:            resourceLimitsTotal,
		resourceLimitsJob:              resourceLimitsJob,
		resourceRequirementsJobDefault: resourceRequirementsJobDefault,
	}, nil
}

// tells you if the given requirements are too much for this capacity manager
// we fill in defaults along the way and return the "processed version"
// to ever run - this is based on the "resourceLimitsJob" not the total
// because we might be busy now but could run the job later
func (manager *CapacityManager) FilterRequirements(requirements ResourceUsageData) (bool, ResourceUsageData) {
	if requirements.CPU <= 0 {
		requirements.CPU = manager.resourceRequirementsJobDefault.CPU
	}
	if requirements.Memory <= 0 {
		requirements.Memory = manager.resourceRequirementsJobDefault.Memory
	}
	if requirements.Disk <= 0 {
		requirements.Disk = manager.resourceRequirementsJobDefault.Disk
	}
	if requirements.GPU <= 0 {
		requirements.GPU = manager.resourceRequirementsJobDefault.GPU
	}
	isOk := checkResourceUsage(requirements, manager.resourceLimitsJob)
	return isOk, requirements
}

func (manager *CapacityManager) GetFreeSpace() ResourceUsageData {
	currentResourceUsage := ResourceUsageData{}

	manager.capacityTracker.ActiveIterator(func(item CapacityManagerItem) {
		currentResourceUsage.CPU += item.Requirements.CPU
		currentResourceUsage.Memory += item.Requirements.Memory
		currentResourceUsage.Disk += item.Requirements.Disk
		currentResourceUsage.GPU += item.Requirements.GPU
	})

	return subtractResourceUsage(currentResourceUsage, manager.resourceLimitsTotal)
}

// get the jobs we have capacity to bid on
// this is done FIFO order from the order jobs have arrived
//   - calculate "remaining resources"
//   - this is total - running
//   - loop over each job in selected queue
//   - if there is enough in the remaining then bid
//   - add each bid on job to the "projected resources"
//   - repeat until project resources >= total resources or no more jobs in queue
func (manager *CapacityManager) GetNextItems() []string {
	// the list of job ids that we have capacity to run
	ids := []string{}

	freeSpace := manager.GetFreeSpace()

	manager.capacityTracker.BacklogIterator(func(item CapacityManagerItem) {
		if checkResourceUsage(item.Requirements, freeSpace) {
			ids = append(ids, item.ID)
			freeSpace = subtractResourceUsage(item.Requirements, freeSpace)
		}
	})

	return ids
}
