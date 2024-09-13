package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	otellog "go.opentelemetry.io/otel/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type EventType string

const (
	JobEventType = "job"
)

type Event struct {
	Type       string
	Properties any
}

// NewEvent creates a new Event.
func NewEvent(eventType string, properties any) *Event {
	return &Event{
		Type:       eventType,
		Properties: properties,
	}
}

// ToLogRecord converts an Event to a LogRecord.
func (e *Event) ToLogRecord() (otellog.Record, error) {
	// Convert the event properties to json
	propertiesJSON, err := json.Marshal(e.Properties)
	if err != nil {
		return otellog.Record{}, err
	}

	record := otellog.Record{}
	record.AddAttributes(
		otellog.String("event", e.Type),
		otellog.String("properties", string(propertiesJSON)),
	)
	record.SetTimestamp(time.Now().UTC())
	record.SetBody(otellog.StringValue(""))
	return record, nil
}

func hashString(in string) string {
	hash := sha256.New()
	hash.Write([]byte(in))
	return hex.EncodeToString(hash.Sum(nil))
}

type JobTerminalEvent struct {
	ID            string `json:"id"`
	NameSet       bool   `json:"name_set"`
	NamespaceHash string `json:"namespace_hash"`
	Type          string `json:"type"`
	Count         int    `json:"count"`
	LabelsCount   int    `json:"labels_count"`
	MetaCount     int    `json:"meta_count"`
	State         string `json:"state"`
	StateMessage  string `json:"state_message"`
	Version       uint64 `json:"version"`
	Revision      uint64 `json:"revision"`
	CreateTime    int64  `json:"create_time"`
	ModifyTime    int64  `json:"modify_time"`

	TaskNameHash         string   `json:"task_name_hash"`
	TaskEngineType       string   `json:"task_engine_type"`
	TaskPublisherType    string   `json:"task_publisher_type"`
	TaskEnvVarCount      int      `json:"task_env_var_count"`
	TaskMetaCount        int      `json:"task_meta_count"`
	TaskInputSourceTypes []string `json:"task_input_source_types"`
	TaskResultPathCount  int      `json:"task_result_path_count"`
	TaskCPUUnits         float64  `json:"task_cpu_units"`
	TaskMemoryBytes      uint64   `json:"task_memory_bytes"`
	TaskDiskBytes        uint64   `json:"task_disk_bytes"`
	TaskGPUCount         uint64   `json:"task_gpu_count"`
	TaskNetworkType      string   `json:"task_network_type"`
	TaskDomainsCount     int      `json:"task_domains_count"`
	TaskExecutionTimeout int64    `json:"task_execution_timeout"`
	TaskQueueTimeout     int64    `json:"task_queue_timeout"`
	TaskTotalTimeout     int64    `json:"task_total_timeout"`
}

func NewJobTerminalEvent(j models.Job) *Event {
	t := j.Task()
	taskInputTypes := make([]string, len(t.InputSources))
	for i, s := range t.InputSources {
		taskInputTypes[i] = s.Source.Type
	}
	// if we can't parse the resources use zero
	taskResources, err := t.ResourcesConfig.ToResources()
	if err != nil {
		taskResources = &models.Resources{
			CPU:    0,
			Memory: 0,
			Disk:   0,
			GPU:    0,
			GPUs:   nil,
		}
	}
	terminalJobEvent := JobTerminalEvent{
		ID:                   j.ID,
		NameSet:              j.ID != j.Name,
		NamespaceHash:        hashString(j.Namespace),
		Type:                 j.Type,
		Count:                j.Count,
		LabelsCount:          len(j.Labels),
		MetaCount:            len(j.Meta),
		State:                j.State.StateType.String(),
		StateMessage:         j.State.Message,
		Version:              j.Version,
		Revision:             j.Revision,
		CreateTime:           j.CreateTime,
		ModifyTime:           j.ModifyTime,
		TaskNameHash:         hashString(t.Name),
		TaskEngineType:       t.Engine.Type,
		TaskPublisherType:    t.Publisher.Type,
		TaskEnvVarCount:      len(t.Env),
		TaskMetaCount:        len(t.Meta),
		TaskInputSourceTypes: taskInputTypes,
		TaskResultPathCount:  len(t.ResultPaths),
		TaskCPUUnits:         taskResources.CPU,
		TaskMemoryBytes:      taskResources.Memory,
		TaskDiskBytes:        taskResources.Disk,
		TaskGPUCount:         taskResources.GPU,
		TaskNetworkType:      t.Network.Type.String(),
		TaskDomainsCount:     len(t.Network.Domains),
		TaskExecutionTimeout: t.Timeouts.ExecutionTimeout,
		TaskQueueTimeout:     t.Timeouts.QueueTimeout,
		TaskTotalTimeout:     t.Timeouts.TotalTimeout,
	}

	return NewEvent(JobEventType, terminalJobEvent)
}
