package analytics

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// SubmitJobEventType is the event type for a job that has been submitted to an orchestrator.
const SubmitJobEventType = "bacalhau.job_v1.submit"

type SubmitJobEvent struct {
	JobID string `json:"job_id"`

	NameSet       bool   `json:"name_set"`
	NamespaceHash string `json:"namespace_hash"`

	Type        string `json:"type"`
	Count       int    `json:"count"`
	LabelsCount int    `json:"labels_count"`
	MetaCount   int    `json:"meta_count"`

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

	Resources Resource `json:"resources,omitempty"`

	TaskNetworkType      string `json:"task_network_type"`
	TaskDomainsCount     int    `json:"task_domains_count"`
	TaskExecutionTimeout int64  `json:"task_execution_timeout"`
	TaskQueueTimeout     int64  `json:"task_queue_timeout"`
	TaskTotalTimeout     int64  `json:"task_total_timeout"`

	Warnings []string `json:"warnings"`
	Error    string   `json:"error"`
}

func NewSubmitJobEvent(j models.Job, warnings ...string) SubmitJobEvent {
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
	return SubmitJobEvent{
		JobID:                j.ID,
		NameSet:              j.ID != j.Name,
		NamespaceHash:        hashString(j.Namespace),
		Type:                 j.Type,
		Count:                j.Count,
		LabelsCount:          len(j.Labels),
		MetaCount:            len(j.Meta),
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
		Resources:            resource,
		TaskNetworkType:      t.Network.Type.String(),
		TaskDomainsCount:     len(t.Network.Domains),
		TaskExecutionTimeout: t.Timeouts.ExecutionTimeout,
		TaskQueueTimeout:     t.Timeouts.QueueTimeout,
		TaskTotalTimeout:     t.Timeouts.TotalTimeout,
		Warnings:             warnings,
	}
}
