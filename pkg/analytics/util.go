package analytics

import (
	otellog "go.opentelemetry.io/otel/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func makeJobAttributes(j models.Job) []otellog.KeyValue {
	return []otellog.KeyValue{
		otellog.String("id", j.ID),
		otellog.String("name", j.Name),
		otellog.String("namespace", j.Namespace),
		otellog.String("type", j.Type),
		otellog.Int("count", j.Count),
		// TODO(forrest): consider collecting constraints, meta, and labels
		otellog.String("state", j.State.StateType.String()),
		otellog.String("state_message", j.State.Message),
		otellog.Int64("version", int64(j.Version)),
		otellog.Int64("revision", int64(j.Revision)),
		otellog.Int64("create_time", j.CreateTime),
		otellog.Int64("modified_time", j.ModifyTime),
	}
}

func makeTaskAttributes(t *models.Task) []otellog.KeyValue {
	inputTypes := make([]otellog.Value, len(t.InputSources))
	for i, s := range t.InputSources {
		inputTypes[i] = otellog.StringValue(s.Source.Type)
	}

	return []otellog.KeyValue{
		otellog.String("task_name", t.Name),
		otellog.String("task_engine", t.Engine.Type),
		otellog.String("task_publisher", t.Engine.Type),
		otellog.Slice("task_inputs", inputTypes...),
		otellog.Int("task_env_count", len(t.Env)),
		otellog.Int("task_meta_count", len(t.Meta)),
		otellog.String("task_cpu", t.ResourcesConfig.CPU),
		otellog.String("task_memory", t.ResourcesConfig.Memory),
		otellog.String("task_disk", t.ResourcesConfig.Disk),
		otellog.String("task_gpu", t.ResourcesConfig.GPU),
		otellog.String("task_network_type", t.Network.Type.String()),
		otellog.Int64("task_timout_execution", t.Timeouts.ExecutionTimeout),
		otellog.Int64("task_timout_queue", t.Timeouts.QueueTimeout),
		otellog.Int64("task_timout_total", t.Timeouts.TotalTimeout),
	}
}
