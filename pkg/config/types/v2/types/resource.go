package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ResourceConfig represents allocated computing resources.
// The resource values are specified in Kubernetes format.
type ResourceConfig struct {
	// CPU specifies the amount of CPU allocated, in Kubernetes format (e.g., "100m" for 100 millicores).
	CPU string
	// Memory specifies the amount of memory allocated, in Kubernetes format (e.g., "1Gi" for 1 Gibibyte).
	Memory string
	// Disk specifies the amount of disk space allocated, in Kubernetes format (e.g., "10Gi" for 10 Gibibytes).
	Disk string
	// GPU specifies the amount of GPU resources allocated, in Kubernetes format (e.g., "1" for 1 GPU).
	GPU string
}

func (r *ResourceConfig) IsZero() bool {
	return r.CPU == "" && r.Memory == "" && r.Disk == "" && r.GPU == ""
}

// Normalize normalizes the resources
func (r *ResourceConfig) Normalize() {
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

type ResourceScalerConfig struct {
	// CPU specifies the amount of CPU allocated as a percentage.
	CPU PercentageValue `yaml:"CPU"`
	// Memory specifies the amount of Memory allocated as a percentage.
	Memory PercentageValue `yaml:"Memory"`
	// Disk specifies the amount of Disk space allocated as a percentage.
	Disk PercentageValue `yaml:"Disk"`
	// GPU specifies the amount of GPU allocated as a percentage.
	GPU PercentageValue `yaml:"GPU"`
}

func (r *ResourceScalerConfig) IsZero() bool {
	return r.CPU == "" && r.Memory == "" && r.Disk == "" && r.GPU == ""
}

func (r *ResourceScalerConfig) Validate() error {
	fields := map[string]PercentageValue{
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

type PercentageValue string

func (pv PercentageValue) Parse() (float64, error) {
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
		return 0, fmt.Errorf("percentage must be a whole number")
	}

	if value < 1 || value > 100 {
		return 0, fmt.Errorf("percentage must be between 1 and 100")
	}

	return value / 100, nil
}
