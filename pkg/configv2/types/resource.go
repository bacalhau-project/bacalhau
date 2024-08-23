package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ResourceScaler struct {
	// CPU specifies the amount of CPU allocated as a percentage.
	CPU Percentage `yaml:"CPU,omitempty"`
	// Memory specifies the amount of Memory allocated as a percentage.
	Memory Percentage `yaml:"Memory,omitempty"`
	// Disk specifies the amount of Disk space allocated as a percentage.
	Disk Percentage `yaml:"Disk,omitempty"`
	// GPU specifies the amount of GPU allocated as a percentage.
	GPU Percentage `yaml:"GPU,omitempty"`
}

func (r ResourceScaler) IsZero() bool {
	return r.CPU == "" && r.Memory == "" && r.Disk == "" && r.GPU == ""
}

func (r ResourceScaler) Validate() error {
	fields := map[string]Percentage{
		"CPU":    r.CPU,
		"Memory": r.Memory,
		"Disk":   r.Disk,
		"GPU":    r.GPU,
	}

	for field, value := range fields {
		if value != "" {
			if _, err := value.Parse(); err != nil {
				return fmt.Errorf("invalid %s percentage: %w", field, err)
			}
		}
	}
	return nil
}

func (r ResourceScaler) Scale(resources models.Resources) (models.Resources, error) {
	if err := r.Validate(); err != nil {
		return models.Resources{}, fmt.Errorf("invalid allocated capacity config: %w", err)
	}
	out := models.Resources{}
	if r.CPU != "" {
		cpuScale, err := r.CPU.Parse()
		if err != nil {
			return models.Resources{}, fmt.Errorf("invalid CPU allocation: %w", err)
		}
		out.CPU = resources.CPU * cpuScale
	}

	if r.Memory != "" {
		memoryScale, err := r.Memory.Parse()
		if err != nil {
			return models.Resources{}, fmt.Errorf("invalid Memory allocation: %w", err)
		}
		out.Memory = uint64(float64(resources.Memory) * memoryScale)
	}

	if r.Disk != "" {
		diskScale, err := r.Disk.Parse()
		if err != nil {
			return models.Resources{}, fmt.Errorf("invalid Disk allocation: %w", err)
		}
		out.Disk = uint64(float64(resources.Disk) * diskScale)
	}

	// TODO(forrest): at present it is unclear how/if GPU resources should be scaled.
	// Currently, tasks denote the number of GPU units required.
	// For example, in a task spec, '2' would signify a requirement of 2 GPU units.
	// If the scaler is .8, then we'd get 1.6 GPUs here, which doesn't work.
	/*
		if r.GPU != "" {
			gpuScale, err := r.GPU.Parse()
			if err != nil {
				return models.Resources{}, fmt.Errorf("invalid GPU allocation")
			}
			out.GPU = uint64(float64(resources.GPU) * gpuScale)
		}
	*/

	return out, nil
}

type Percentage string

func (pv Percentage) Parse() (float64, error) {
	// if the percentage is empty then it means the user didn't provide a config
	// value, and we shouldn't scale. Returning 1 will result in no scaling.
	if pv == "" {
		return 1, nil
	}
	s := strings.TrimSpace(string(pv))
	s = strings.TrimSuffix(s, "%")

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid percentage format: %w", err)
	}

	if value != float64(int(value)) {
		return 0, fmt.Errorf("percentage must be a whole number. receieved %q", pv)
	}

	if value < 1 || value > 100 {
		return 0, fmt.Errorf(`percentage must be between 1%% and 100%%. receieved %q`, pv)
	}

	return value / 100, nil
}
