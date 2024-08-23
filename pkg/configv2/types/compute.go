package types

import (
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type Compute struct {
	Enabled           bool              `yaml:"Enabled,omitempty"`
	Orchestrators     []string          `yaml:"Orchestrators,omitempty"`
	TLS               TLS               `yaml:"TLS,omitempty"`
	Heartbeat         Heartbeat         `yaml:"Heartbeat,omitempty"`
	Labels            map[string]string `yaml:"Labels,omitempty"`
	AllocatedCapacity ResourceScaler    `yaml:"AllocatedCapacity,omitempty"`
	Volumes           []Volume          `yaml:"Volumes,omitempty"`
}

func (c Compute) Validate() error {
	if c.Enabled {
		if err := validate.IsNotEmpty(c.Orchestrators, "at least one compute orchestrator is required"); err != nil {
			return err
		}
		for i, o := range c.Orchestrators {
			if err := validateURL(o, "nats", ""); err != nil {
				return fmt.Errorf("compute orchestrator %q at index %d is invalid: %w", o, i, err)
			}
		}
		if err := validateFields(c); err != nil {
			return err
		}
	}
	return nil
}

type Heartbeat struct {
	InfoUpdateInterval     Duration `yaml:"InfoUpdateInterval,omitempty"`
	ResourceUpdateInterval Duration `yaml:"ResourceUpdateInterval,omitempty"`
	Interval               Duration `yaml:"Interval,omitempty"`
}

func (c Heartbeat) Validate() error {
	if c.Interval < 0 {
		return fmt.Errorf("heart beat interval cannot be less than zero. received: %d", c.Interval)
	}
	if c.ResourceUpdateInterval < 0 {
		return fmt.Errorf("heart beat resource update interval cannot be less than zero. received: %d", c.ResourceUpdateInterval)
	}
	if c.InfoUpdateInterval < 0 {
		return fmt.Errorf("heart beat info update interval cannot be less than zero. received: %d", c.InfoUpdateInterval)
	}

	return nil
}

type Volume struct {
	Name      string `yaml:"Name,omitempty"`
	Path      string `yaml:"Path,omitempty"`
	ReadWrite bool   `yaml:"Write,omitempty"`
}

func (c Volume) Validate() error {
	if c.Name != "" && c.Path == "" {
		return fmt.Errorf("volume must have path when name is provided")
	}
	if _, err := os.Stat(c.Path); err != nil {
		return fmt.Errorf("volume with path %q is invalid. path not readable: %w", c.Path, err)
	}
	return nil
}
