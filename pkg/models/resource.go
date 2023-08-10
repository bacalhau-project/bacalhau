package models

import (
	"errors"
	"fmt"

	"github.com/c2h5oh/datasize"
	"github.com/hashicorp/go-multierror"
)

type Resources struct {
	// CPU units
	CPU float64
	// Memory in bytes
	Memory uint64
	// Disk in bytes
	Disk uint64
	// GPU units
	GPU uint64
}

// Validate returns an error if the resources are invalid
func (r *Resources) Validate() error {
	if r == nil {
		return errors.New("missing resources")
	}
	var mErr multierror.Error
	if r.CPU <= 0 {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("invalid CPU value: %f", r.CPU))
	}
	if r.Memory == 0 {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("invalid memory value: %d", r.Memory))
	}
	return mErr.ErrorOrNil()
}

// Add returns the sum of the resources
func (r *Resources) Add(other *Resources) *Resources {
	return &Resources{
		CPU:    r.CPU + other.CPU,
		Memory: r.Memory + other.Memory,
		Disk:   r.Disk + other.Disk,
		GPU:    r.GPU + other.GPU,
	}
}

// Merge merges the resources, preferring the current resources
func (r *Resources) Merge(other *Resources) *Resources {
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
	return newR
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

func (r *Resources) LessThanEq(other Resources) bool {
	return r.CPU <= other.CPU && r.Memory <= other.Memory && r.Disk <= other.Disk && r.GPU <= other.GPU
}

func (r *Resources) IsZero() bool {
	return r.CPU == 0 && r.Memory == 0 && r.Disk == 0 && r.GPU == 0
}

// return string representation of ResourceUsageData
func (r *Resources) String() string {
	mem := datasize.ByteSize(r.Memory)
	disk := datasize.ByteSize(r.Disk)
	return fmt.Sprintf("{CPU: %f, Memory: %s, Disk: %s, GPU: %d}", r.CPU, mem.HR(), disk.HR(), r.GPU)
}

// AllocatedTaskResources is the resources allocated to a task.
// For now it is the same as Resources, but in the future it may include
// additional information about the actual resources allocated, such as the
// device ID of a GPU.
type AllocatedTaskResources Resources

func (a *AllocatedTaskResources) Copy() *AllocatedTaskResources {
	if a == nil {
		return nil
	}
	newR := new(AllocatedTaskResources)
	*newR = *a
	return newR
}

// AllocatedResources is the set of resources to be used by an execution, which
// maybe be resources allocated to a single task or a set of tasks in the future.
type AllocatedResources struct {
	Tasks map[string]*AllocatedTaskResources
}

func (a *AllocatedResources) Copy() *AllocatedResources {
	if a == nil {
		return a
	}
	tasks := make(map[string]*AllocatedTaskResources)
	for k, v := range a.Tasks {
		tasks[k] = v.Copy()
	}
	return &AllocatedResources{
		Tasks: tasks,
	}
}
