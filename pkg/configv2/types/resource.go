package types

import (
	"fmt"
	"strconv"
	"strings"
)

// Resource represents allocated computing resources.
// The resource values are specified in Kubernetes format.
type Resource struct {
	// CPU specifies the amount of CPU allocated, in Kubernetes format (e.g., "100m" for 100 millicores).
	CPU string `yaml:"CPU,omitempty"`
	// Memory specifies the amount of memory allocated, in Kubernetes format (e.g., "1Gi" for 1 Gibibyte).
	Memory string `yaml:"Memory,omitempty"`
	// Disk specifies the amount of disk space allocated, in Kubernetes format (e.g., "10Gi" for 10 Gibibytes).
	Disk string `yaml:"Disk,omitempty"`
	// GPU specifies the amount of GPU resources allocated, in Kubernetes format (e.g., "1" for 1 GPU).
	GPU string `yaml:"GPU,omitempty"`
}

func (r *Resource) IsZero() bool {
	return r.CPU == "" && r.Memory == "" && r.Disk == "" && r.GPU == ""
}

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
