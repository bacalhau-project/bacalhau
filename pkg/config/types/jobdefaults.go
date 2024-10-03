package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobDefaults struct {
	Batch   BatchJobDefaultsConfig       `yaml:"Batch,omitempty" json:"Batch,omitempty"`
	Ops     BatchJobDefaultsConfig       `yaml:"Ops,omitempty" json:"Ops,omitempty"`
	Daemon  LongRunningJobDefaultsConfig `yaml:"Daemon,omitempty" json:"Daemon,omitempty"`
	Service LongRunningJobDefaultsConfig `yaml:"Service,omitempty" json:"Service,omitempty"`
}

type BatchJobDefaultsConfig struct {
	// Priority specifies the default priority allocated to a batch or ops job.
	// This value is used when the job hasn't explicitly set its priority requirement.
	Priority int                    `yaml:"Priority,omitempty" json:"Priority,omitempty"`
	Task     BatchTaskDefaultConfig `yaml:"Task,omitempty" json:"Task,omitempty"`
}

type BatchTaskDefaultConfig struct {
	Resources ResourcesConfig   `yaml:"Resources,omitempty" json:"Resources,omitempty"`
	Publisher models.SpecConfig `yaml:"Publisher,omitempty"`
	Timeouts  TaskTimeoutConfig `yaml:"Timeouts,omitempty" json:"Timeouts,omitempty"`
}

type ResourcesConfig struct {
	// CPU specifies the default amount of CPU allocated to a task.
	// It uses Kubernetes resource string format (e.g., "100m" for 0.1 CPU cores).
	// This value is used when the task hasn't explicitly set its CPU requirement.
	CPU string `yaml:"CPU,omitempty" json:"CPU,omitempty"`

	// Memory specifies the default amount of memory allocated to a task.
	// It uses Kubernetes resource string format (e.g., "256Mi" for 256 mebibytes).
	// This value is used when the task hasn't explicitly set its memory requirement.
	Memory string `yaml:"Memory,omitempty" json:"Memory,omitempty"`

	// Disk specifies the default amount of disk space allocated to a task.
	// It uses Kubernetes resource string format (e.g., "1Gi" for 1 gibibyte).
	// This value is used when the task hasn't explicitly set its disk space requirement.
	Disk string `yaml:"Disk,omitempty" json:"Disk,omitempty"`

	// GPU specifies the default number of GPUs allocated to a task.
	// It uses Kubernetes resource string format (e.g., "1" for 1 GPU).
	// This value is used when the task hasn't explicitly set its GPU requirement.
	GPU string `yaml:"GPU,omitempty" json:"GPU,omitempty"`
}

// FromModelsResourceConfig converts a models.ResourcesConfig to a ResourcesConfig.
func FromModelsResourceConfig(r models.ResourcesConfig) ResourcesConfig {
	return ResourcesConfig{
		CPU:    r.CPU,
		Memory: r.Memory,
		Disk:   r.Disk,
		GPU:    r.GPU,
	}
}

type TaskTimeoutConfig struct {
	// TotalTimeout is the maximum total time allowed for a task
	TotalTimeout Duration `yaml:"TotalTimeout,omitempty" json:"TotalTimeout,omitempty"`
	// ExecutionTimeout is the maximum time allowed for task execution
	ExecutionTimeout Duration `yaml:"ExecutionTimeout,omitempty" json:"ExecutionTimeout,omitempty"`
}

type LongRunningJobDefaultsConfig struct {
	// Priority specifies the default priority allocated to a service or daemon job.
	// This value is used when the job hasn't explicitly set its priority requirement.
	Priority int                          `yaml:"Priority,omitempty" json:"Priority,omitempty"`
	Task     LongRunningTaskDefaultConfig `yaml:"Task,omitempty" json:"Task,omitempty"`
}

type LongRunningTaskDefaultConfig struct {
	Resources ResourcesConfig `yaml:"Resources,omitempty" json:"Resources,omitempty"`
}
