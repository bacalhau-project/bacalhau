package v1beta1

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// a record for the "amount" of compute resources an entity has / can consume / is using

type ResourceUsageConfig struct {
	// https://github.com/BTBurke/k8sresource string
	CPU string `json:"CPU,omitempty"`
	// github.com/c2h5oh/datasize string
	Memory string `json:"Memory,omitempty"`
	// github.com/c2h5oh/datasize string

	Disk string `json:"Disk,omitempty"`
	GPU  string `json:"GPU"` // unsigned integer string

}

// these are the numeric values in bytes for ResourceUsageConfig
type ResourceUsageData struct {
	// cpu units
	CPU float64 `json:"CPU,omitempty" example:"9.600000000000001"`
	// bytes
	Memory uint64 `json:"Memory,omitempty" example:"27487790694"`
	// bytes
	Disk uint64 `json:"Disk,omitempty" example:"212663867801"`
	GPU  uint64 `json:"GPU,omitempty" example:"1"` //nolint:lll // Support whole GPUs only, like https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/
}

func (r ResourceUsageData) Add(other ResourceUsageData) ResourceUsageData {
	return ResourceUsageData{
		CPU:    r.CPU + other.CPU,
		Memory: r.Memory + other.Memory,
		Disk:   r.Disk + other.Disk,
		GPU:    r.GPU + other.GPU,
	}
}

func (r ResourceUsageData) Sub(other ResourceUsageData) ResourceUsageData {
	usage := ResourceUsageData{
		CPU:    r.CPU - other.CPU,
		Memory: r.Memory - other.Memory,
		Disk:   r.Disk - other.Disk,
		GPU:    r.GPU - other.GPU,
	}

	if r.LessThan(other) {
		log.Warn().Msgf("Subtracting larger resource usage %s from %s. Replacing negative values with zeros",
			other.String(), r.String())
		if other.CPU > r.CPU {
			usage.CPU = 0
		}
		if other.Memory > r.Memory {
			usage.Memory = 0
		}
		if other.Disk > r.Disk {
			usage.Disk = 0
		}
		if other.GPU > r.GPU {
			usage.GPU = 0
		}
	}

	return usage
}

func (r ResourceUsageData) Multi(factor float64) ResourceUsageData {
	return ResourceUsageData{
		CPU:    r.CPU * factor,
		Memory: uint64(float64(r.Memory) * factor),
		Disk:   uint64(float64(r.Disk) * factor),
		GPU:    uint64(float64(r.GPU) * factor),
	}
}

func (r ResourceUsageData) Intersect(other ResourceUsageData) ResourceUsageData {
	if r.CPU <= 0 {
		r.CPU = other.CPU
	}
	if r.Memory <= 0 {
		r.Memory = other.Memory
	}
	if r.Disk <= 0 {
		r.Disk = other.Disk
	}
	if r.GPU <= 0 {
		r.GPU = other.GPU
	}

	return r
}

func (r ResourceUsageData) Max(other ResourceUsageData) ResourceUsageData {
	if r.CPU < other.CPU {
		r.CPU = other.CPU
	}
	if r.Memory < other.Memory {
		r.Memory = other.Memory
	}
	if r.Disk < other.Disk {
		r.Disk = other.Disk
	}
	if r.GPU < other.GPU {
		r.GPU = other.GPU
	}

	return r
}

func (r ResourceUsageData) LessThan(other ResourceUsageData) bool {
	return r.CPU < other.CPU && r.Memory < other.Memory && r.Disk < other.Disk && r.GPU < other.GPU
}

func (r ResourceUsageData) LessThanEq(other ResourceUsageData) bool {
	return r.CPU <= other.CPU && r.Memory <= other.Memory && r.Disk <= other.Disk && r.GPU <= other.GPU
}

func (r ResourceUsageData) IsZero() bool {
	return r.CPU == 0 && r.Memory == 0 && r.Disk == 0 && r.GPU == 0
}

// return string representation of ResourceUsageData
func (r ResourceUsageData) String() string {
	return fmt.Sprintf("{CPU: %f, Memory: %d, Disk: %d, GPU: %d}", r.CPU, r.Memory, r.Disk, r.GPU)
}

type ResourceUsageProfile struct {
	// how many resources does the job want to consume
	Job ResourceUsageData `json:"Job,omitempty"`
	// how many resources is the system currently using
	SystemUsing ResourceUsageData `json:"SystemUsing,omitempty"`
	// what is the total amount of resources available to the system
	SystemTotal ResourceUsageData `json:"SystemTotal,omitempty"`
}
