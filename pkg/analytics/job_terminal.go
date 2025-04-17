package analytics

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// TerminalJobEventType is the event type for a job that has reached a terminal state.
const TerminalJobEventType = "bacalhau.job_v1.terminal"

type JobTerminalEvent struct {
	JobID string `json:"job_id"`

	NameSet       bool   `json:"name_set"`
	NamespaceHash string `json:"namespace_hash"`

	Type        string `json:"type"`
	Count       int    `json:"count"`
	LabelsCount int    `json:"labels_count"`
	MetaCount   int    `json:"meta_count"`

	State string `json:"state"`

	Version    uint64    `json:"version"`
	Revision   uint64    `json:"revision"`
	CreateTime time.Time `json:"create_time"`
	ModifyTime time.Time `json:"modify_time"`

	TaskNameHash         string   `json:"task_name_hash"`
	TaskEngineType       string   `json:"task_engine_type"`
	TaskPublisherType    string   `json:"task_publisher_type"`
	TaskEnvVarCount      int      `json:"task_env_var_count"`
	TaskMetaCount        int      `json:"task_meta_count"`
	TaskInputSourceTypes []string `json:"task_input_source_types"`
	TaskResultPathCount  int      `json:"task_result_path_count"`
	// TaskDockerImage captures Docker image information, only populated when TaskEngineType is "docker".
	// For non-trusted images (not from bacalhau or expanso), the image name is hashed for privacy.
	TaskDockerImage string `json:"task_docker_image,omitempty"`

	Resources Resource `json:"resources,omitempty"`

	TaskNetworkType      string `json:"task_network_type"`
	TaskDomainsCount     int    `json:"task_domains_count"`
	TaskExecutionTimeout int64  `json:"task_execution_timeout"`
	TaskQueueTimeout     int64  `json:"task_queue_timeout"`
	TaskTotalTimeout     int64  `json:"task_total_timeout"`
}

func NewJobTerminalEvent(j models.Job) *Event {
	t := j.Task()
	taskInputTypes := make([]string, len(t.InputSources))
	for i, s := range t.InputSources {
		taskInputTypes[i] = s.Source.Type
	}
	// if we can't parse the resources use zero
	var resource Resource
	taskResources, err := t.ResourcesConfig.ToResources()
	if err != nil {
		resource = Resource{
			CPUUnits:    0,
			MemoryBytes: 0,
			DiskBytes:   0,
			GPUCount:    0,
			GPUTypes:    nil,
		}
	} else {
		gpuTypes := make([]GPUInfo, len(taskResources.GPUs))
		for i, gpu := range taskResources.GPUs {
			gpuTypes[i] = GPUInfo{
				Name:   gpu.Name,
				Vendor: string(gpu.Vendor),
			}
		}
		resource = Resource{
			CPUUnits:    taskResources.CPU,
			MemoryBytes: taskResources.Memory,
			DiskBytes:   taskResources.Disk,
			GPUCount:    taskResources.GPU,
			GPUTypes:    gpuTypes,
		}
	}
	terminalJobEvent := JobTerminalEvent{
		JobID:                j.ID,
		NameSet:              j.ID != j.Name,
		NamespaceHash:        hashString(j.Namespace),
		Type:                 j.Type,
		Count:                j.Count,
		LabelsCount:          len(j.Labels),
		MetaCount:            len(j.Meta),
		State:                j.State.StateType.String(),
		Version:              j.Version,
		Revision:             j.Revision,
		CreateTime:           time.Unix(0, j.CreateTime).UTC(),
		ModifyTime:           time.Unix(0, j.ModifyTime).UTC(),
		TaskNameHash:         hashString(t.Name),
		TaskEngineType:       t.Engine.Type,
		TaskPublisherType:    t.Publisher.Type,
		TaskEnvVarCount:      len(t.Env),
		TaskMetaCount:        len(t.Meta),
		TaskInputSourceTypes: taskInputTypes,
		TaskResultPathCount:  len(t.ResultPaths),
		TaskDockerImage:      GetDockerImageTelemetry(t.Engine),
		Resources:            resource,
		TaskNetworkType:      t.Network.Type.String(),
		TaskDomainsCount:     len(t.Network.Domains),
		TaskExecutionTimeout: t.Timeouts.ExecutionTimeout,
		TaskQueueTimeout:     t.Timeouts.QueueTimeout,
		TaskTotalTimeout:     t.Timeouts.TotalTimeout,
	}

	return NewEvent(TerminalJobEventType, terminalJobEvent)
}
