package publicapi

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
)

type HandleJobEventWrapper struct {
	Server *RequesterAPIServer
}

func (h *HandleJobEventWrapper) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	return h.Server.HandleJobEvent(ctx, v1beta2.JobEvent{
		JobID:        event.JobID,
		ExecutionID:  event.ExecutionID,
		SourceNodeID: event.SourceNodeID,
		TargetNodeID: event.TargetNodeID,
		EventName:    v1beta2.JobEventType(event.EventName),
		Status:       event.Status,
		EventTime:    event.EventTime,
	})
}

type DebugInfoProviderWrapper struct {
	Provider model.DebugInfoProvider
}

func (d *DebugInfoProviderWrapper) GetDebugInfo(ctx context.Context) (v1beta2.DebugInfo, error) {
	info, err := d.Provider.GetDebugInfo(ctx)
	if err != nil {
		return v1beta2.DebugInfo{}, err
	}
	return v1beta2.DebugInfo{
		Component: info.Component,
		Info:      info.Info,
	}, nil
}
