package models

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type ResourcesConfig struct {
	// CPU https://github.com/BTBurke/k8sresource string
	CPU string `json:"CPU,omitempty"`
	// Memory github.com/dustin/go-humanize string
	Memory string `json:"Memory,omitempty"`
	// Memory github.com/dustin/go-humanize string
	Disk string `json:"Disk,omitempty"`
	GPU  string `json:"GPU,omitempty"`
}

// Normalize normalizes the resources
func (r *ResourcesConfig) Normalize() {
	if r == nil {
		return
	}
	sanitizeResourceString := func(s string) string {
		// lower case and remove spaces
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\n", "")
		return s
	}

	r.CPU = sanitizeResourceString(r.CPU)
	r.Memory = sanitizeResourceString(r.Memory)
	r.Disk = sanitizeResourceString(r.Disk)
	r.GPU = sanitizeResourceString(r.GPU)
}

// Copy returns a deep copy of the resources
func (r *ResourcesConfig) Copy() *ResourcesConfig {
	if r == nil {
		return nil
	}
	newR := new(ResourcesConfig)
	*newR = *r
	return newR
}

// Validate returns an error if the resources are invalid
func (r *ResourcesConfig) Validate() error {
	if r == nil {
		return nil
	}
	resources, err := r.ToResources()
	if err != nil {
		return err
	}
	return resources.Validate()
}

// ToResources converts the resources config to resources
func (r *ResourcesConfig) ToResources() (*Resources, error) {
	if r == nil {
		return nil, errors.New("missing resources")
	}
	r.Normalize()
	var mErr error
	res := &Resources{}

	if r.CPU != "" {
		cpu, err := k8sresource.NewCPUFromString(r.CPU)
		if err != nil {
			mErr = errors.Join(mErr, fmt.Errorf("invalid CPU value: %s", r.CPU))
		}
		res.CPU = cpu.ToFloat64()
	}
	if r.Memory != "" {
		mem, err := humanize.ParseBytes(r.Memory)
		if err != nil {
			mErr = errors.Join(mErr, fmt.Errorf("invalid memory value: %s", r.Memory))
		}
		res.Memory = mem
	}
	if r.Disk != "" {
		disk, err := humanize.ParseBytes(r.Disk)
		if err != nil {
			mErr = errors.Join(mErr, fmt.Errorf("invalid disk value: %s", r.Disk))
		}
		res.Disk = disk
	}
	if r.GPU != "" {
		gpu, err := strconv.ParseUint(r.GPU, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid GPU value: %s", r.GPU)
		}
		res.GPU = gpu
	}

	return res, mErr
}

type GPUVendor string

const (
	GPUVendorNvidia GPUVendor = "NVIDIA"
	GPUVendorAMDATI GPUVendor = "AMD/ATI"
	GPUVendorIntel  GPUVendor = "Intel"
)

type GPU struct {
	// Self-reported index of the device in the system
	Index uint64
	// Model name of the GPU e.g. Tesla T4
	Name string
	// Maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor
	// Total GPU memory in mebibytes (MiB)
	Memory uint64
	// PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string
}

// Less compares this GPU with another for sorting/ordering purposes
// The comparison order is: Index, Name, Vendor, Memory, PCIAddress
func (g GPU) Less(other GPU) bool {
	if g.Index != other.Index {
		return g.Index < other.Index
	}
	if g.Name != other.Name {
		return g.Name < other.Name
	}
	if g.Vendor != other.Vendor {
		return g.Vendor < other.Vendor
	}
	if g.Memory != other.Memory {
		return g.Memory < other.Memory
	}
	return g.PCIAddress < other.PCIAddress
}

type Resources struct {
	// CPU units
	CPU float64 `json:"CPU,omitempty"`
	// Memory in bytes
	Memory uint64 `json:"Memory,omitempty"`
	// Disk in bytes
	Disk uint64 `json:"Disk,omitempty"`
	// GPU units
	GPU uint64 `json:"GPU,omitempty"`
	// GPU details
	GPUs []GPU `json:"GPUs,omitempty"`
}

// Copy returns a deep copy of the resources
func (r *Resources) Copy() *Resources {
	if r == nil {
		return nil
	}
	newR := new(Resources)
	*newR = *r
	return newR
}

// Validate returns an error if the resources are invalid
func (r *Resources) Validate() error {
	if r == nil {
		return errors.New("missing resources")
	}
	var mErr error
	if r.CPU < 0 {
		mErr = errors.Join(mErr, fmt.Errorf("invalid CPU value: %f", r.CPU))
	}
	//nolint:gosec // G115: GPU count should be always within reasonable bounds
	if len(r.GPUs) > int(r.GPU) {
		// It's not an error for the GPUs specified to be less than the number
		// given by the GPU field – that just signifies that either:
		// - the user is requesting "generic GPUs" without specifying more information
		// - the system knows it has GPUs but no further information about them
		// But the number should always be at least the length of the GPUs array
		mErr = errors.Join(mErr, fmt.Errorf("%d GPUs specified but have details for %d", r.GPU, len(r.GPUs)))
	}
	return mErr
}

// Merge merges the resources, preferring the current resources
func (r *Resources) Merge(other Resources) *Resources {
	newR := r.Copy()
	if newR.CPU <= 0 {
		newR.CPU = other.CPU
	}
	if newR.Memory <= 0 {
		newR.Memory = other.Memory
	}
	if newR.Disk <= 0 {
		newR.Disk = other.Disk
	}
	if newR.GPU <= 0 {
		newR.GPU = other.GPU
	}
	if len(newR.GPUs) <= 0 {
		newR.GPUs = other.GPUs
	}
	return newR
}

// Add returns the sum of the resources
func (r *Resources) Add(other Resources) *Resources {
	return &Resources{
		CPU:    r.CPU + other.CPU,
		Memory: r.Memory + other.Memory,
		Disk:   r.Disk + other.Disk,
		GPU:    r.GPU + other.GPU,
		GPUs:   append(r.GPUs, other.GPUs...),
	}
}

func (r *Resources) Sub(other Resources) *Resources {
	usage := &Resources{
		CPU:    r.CPU - other.CPU,
		Memory: r.Memory - other.Memory,
		Disk:   r.Disk - other.Disk,
		GPU:    r.GPU - other.GPU,
	}

	usage.GPUs, _ = lo.Difference(r.GPUs, other.GPUs)

	// Check for negative values and replace with zeros
	hasNegativeValues := false

	if other.CPU > r.CPU {
		usage.CPU = 0
		hasNegativeValues = true
	}
	if other.Memory > r.Memory {
		usage.Memory = 0
		hasNegativeValues = true
	}
	if other.Disk > r.Disk {
		usage.Disk = 0
		hasNegativeValues = true
	}
	if other.GPU > r.GPU {
		usage.GPU = 0
		hasNegativeValues = true
	}

	// Log once if any negative values were encountered
	if hasNegativeValues {
		log.Warn().Msgf("Subtracting larger resource usage %s from %s. Replaced negative values with zeros",
			other.String(), r.String())
	}

	return usage
}

// Multiply returns the product of the resources
func (r *Resources) Multiply(factor float64) *Resources {
	return &Resources{
		CPU:    r.CPU * factor,
		Memory: uint64(float64(r.Memory) * factor),
		Disk:   uint64(float64(r.Disk) * factor),
		GPU:    uint64(float64(r.GPU) * factor),
	}
}

func (r *Resources) LessThan(other Resources) bool {
	return r.CPU < other.CPU && r.Memory < other.Memory && r.Disk < other.Disk && r.GPU < other.GPU
}

func (r *Resources) LessThanEq(other Resources) bool {
	return r.CPU <= other.CPU && r.Memory <= other.Memory && r.Disk <= other.Disk && r.GPU <= other.GPU
}

func (r *Resources) Max(other Resources) *Resources {
	newR := r.Copy()
	if newR.CPU < other.CPU {
		newR.CPU = other.CPU
	}
	if newR.Memory < other.Memory {
		newR.Memory = other.Memory
	}
	if newR.Disk < other.Disk {
		newR.Disk = other.Disk
	}
	if newR.GPU < other.GPU {
		newR.GPU = other.GPU
	}

	return newR
}

func (r *Resources) IsZero() bool {
	return r.CPU == 0 && r.Memory == 0 && r.Disk == 0 && r.GPU == 0
}

// return string representation of ResourceUsageData
func (r *Resources) String() string {
	mem := humanize.Bytes(r.Memory)
	disk := humanize.Bytes(r.Disk)
	return fmt.Sprintf("{CPU: %.2f, Memory: %s, Disk: %s, GPU: %d}", r.CPU, mem, disk, r.GPU)
}

// AllocatedResources is the set of resources to be used by an execution, which
// maybe be resources allocated to a single task or a set of tasks in the future.
type AllocatedResources struct {
	Tasks map[string]*Resources `json:"Tasks"`
}

func (a *AllocatedResources) Copy() *AllocatedResources {
	if a == nil {
		return a
	}
	tasks := make(map[string]*Resources)
	for k, v := range a.Tasks {
		tasks[k] = v.Copy()
	}
	return &AllocatedResources{
		Tasks: tasks,
	}
}

// Total returns the total resources allocated
func (a *AllocatedResources) Total() *Resources {
	if a == nil {
		return nil
	}
	total := &Resources{}
	for _, task := range a.Tasks {
		total = total.Add(*task)
	}
	return total
}
