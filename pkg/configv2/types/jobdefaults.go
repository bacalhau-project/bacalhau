package types

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobDefaults struct {
	Batch   BatchJobDefaultsConfig       `yaml:"Batch,omitempty"`
	Ops     BatchJobDefaultsConfig       `yaml:"Ops,omitempty"`
	Daemon  LongRunningJobDefaultsConfig `yaml:"Daemon,omitempty"`
	Service LongRunningJobDefaultsConfig `yaml:"Service,omitempty"`
}

type BatchJobDefaultsConfig struct {
	Priority int                    `yaml:"Priority,omitempty"`
	Task     BatchTaskDefaultConfig `yaml:"Task,omitempty"`
}

type LongRunningJobDefaultsConfig struct {
	Priority int                          `yaml:"Priority,omitempty"`
	Task     LongRunningTaskDefaultConfig `yaml:"Task,omitempty"`
}

type BatchTaskDefaultConfig struct {
	Resources ResourcesConfig        `yaml:"Resources,omitempty"`
	Publisher DefaultPublisherConfig `yaml:"Publisher,omitempty"`
	Timeouts  TaskTimeoutConfig      `yaml:"Timeouts,omitempty"`
}

type LongRunningTaskDefaultConfig struct {
	Resources ResourcesConfig        `yaml:"Resources,omitempty"`
	Publisher DefaultPublisherConfig `yaml:"Publisher,omitempty"`
}

type ResourcesConfig struct {
	CPU    string `yaml:"CPU,omitempty"`
	Memory string `yaml:"Memory,omitempty"`
	Disk   string `yaml:"Disk,omitempty"`
	GPU    string `yaml:"GPU,omitempty"`
}

type DefaultPublisherConfig struct {
	Type string `yaml:"Type,omitempty"`
}

func (c DefaultPublisherConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("default publisher type cannot be empty")
	}
	isValidType := false
	for _, expected := range models.PublisherNames {
		if strings.ToLower(c.Type) == strings.ToLower(expected) {
			isValidType = true
		}
	}
	if !isValidType {
		return fmt.Errorf("default publisher type %q unknow. must be one of: %v", c.Type, models.PublisherNames)
	}
	return nil
}
