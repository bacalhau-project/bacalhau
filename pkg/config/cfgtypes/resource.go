package cfgtypes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/dustin/go-humanize"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ResourceScaler struct {
	// CPU specifies the amount of CPU allocated as a percentage.
	CPU ResourceType `yaml:"CPU,omitempty"`
	// Memory specifies the amount of Memory allocated as a percentage.
	Memory ResourceType `yaml:"Memory,omitempty"`
	// Disk specifies the amount of Disk space allocated as a percentage.
	Disk ResourceType `yaml:"Disk,omitempty"`
	// GPU specifies the amount of GPU allocated as a percentage.
	GPU ResourceType `yaml:"GPU,omitempty"`
}

func (s ResourceScaler) IsZero() bool {
	return s.CPU == "" && s.Memory == "" && s.Disk == "" && s.GPU == ""
}

func (s ResourceScaler) ToResource(in models.Resources) (*models.Resources, error) {
	out := new(models.Resources)
	if s.CPU.IsScaler() {
		scalerStr := strings.TrimSuffix(string(s.CPU), "%")
		value, err := strconv.ParseFloat(scalerStr, 64)
		if err != nil {
			return nil, fmt.Errorf("cpu capacity invalid percentage format: %w", err)
		}
		if value < 0 || value > 100 {
			return nil, fmt.Errorf("cpu capacity percentage must be between 0%% and 100%%, got %s", s.CPU)
		}
		value = value / 100
		out.CPU = in.CPU * value
	} else {
		cpu, err := k8sresource.NewCPUFromString(string(s.CPU))
		if err != nil {
			return nil, fmt.Errorf("invalid CPU value %q: %w", s.CPU, err)
		}
		out.CPU = cpu.ToFloat64()
	}

	if s.Memory.IsScaler() {
		scalerStr := strings.TrimSuffix(string(s.Memory), "%")
		value, err := strconv.ParseFloat(scalerStr, 64)
		if err != nil {
			return nil, fmt.Errorf("memory capacity invalid percentage format: %w", err)
		}
		if value < 0 || value > 100 {
			return nil, fmt.Errorf("memory capacity percentage must be between 0%% and 100%%, got %s", s.Memory)
		}
		value = value / 100
		out.Memory = uint64(float64(in.Memory) * value)
	} else {
		mem, err := humanize.ParseBytes(string(s.Memory))
		if err != nil {
			return nil, fmt.Errorf("invalid Memory value %q: %w", s.Memory, err)
		}
		out.Memory = mem
	}

	if s.Disk.IsScaler() {
		scalerStr := strings.TrimSuffix(string(s.Disk), "%")
		value, err := strconv.ParseFloat(scalerStr, 64)
		if err != nil {
			return nil, fmt.Errorf("disk capacity invalid percentage format: %w", err)
		}
		if value < 0 || value > 100 {
			return nil, fmt.Errorf("disk capacity percentage must be between 0%% and 100%%, got %s", s.Disk)
		}
		value = value / 100
		out.Disk = uint64(float64(in.Disk) * value)
	} else {
		disk, err := humanize.ParseBytes(string(s.Disk))
		if err != nil {
			return nil, fmt.Errorf("invalid Disk value %q: %w", s.Disk, err)
		}
		out.Disk = disk
	}

	if s.GPU.IsScaler() {
		scalerStr := strings.TrimSuffix(string(s.GPU), "%")
		value, err := strconv.ParseFloat(scalerStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage format: %w", err)
		}
		if value < 0 || value > 100 {
			return nil, fmt.Errorf("percentage must be between 0%% and 100%%, got %s", s.GPU)
		}
		value = value / 100
		// ensure we never scale a GPU down to zero unless there isn't a GPU
		tmp := float64(in.GPU) * value
		if tmp < 1 && in.GPU >= 1 {
			tmp = 1
		}
		out.GPU = uint64(tmp)
	} else {
		gpu, err := strconv.ParseUint(string(s.GPU), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid GPU value %q: %w", s.GPU, err)
		}
		out.GPU = gpu
	}

	return out, nil
}

type ResourceType string

func (t ResourceType) IsScaler() bool {
	return strings.HasSuffix(string(t), "%")
}
