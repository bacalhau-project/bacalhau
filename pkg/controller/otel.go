package controller

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// endJobContext ends the global lifecycle context for a job.
func (ctrl *Controller) endJobContext(jobID string) {
	ctx := ctrl.getJobContext(jobID)
	trace.SpanFromContext(ctx).End()
	delete(ctrl.jobContexts, jobID)
}

// endJobNodeContext ends the local lifecycle context for a job.
func (ctrl *Controller) endJobNodeContext(jobID string) {
	ctx := ctrl.getJobNodeContext(context.Background(), jobID)
	trace.SpanFromContext(ctx).End()
	delete(ctrl.jobNodeContexts, jobID)
}

// getJobContext returns a context that tracks the global lifecycle of a job
// as it is processed by this and other nodes in the bacalhau network.
func (ctrl *Controller) getJobContext(jobID string) context.Context {
	ctrl.contextMutex.RLock()
	defer ctrl.contextMutex.RUnlock()
	jobCtx, ok := ctrl.jobContexts[jobID]
	if !ok {
		return context.Background() // no lifecycle context yet
	}
	return jobCtx
}

// getJobNodeContext returns a context that tracks the local lifecycle of a
// job as it has been processed by this node.
func (ctrl *Controller) getJobNodeContext(ctx context.Context, jobID string) context.Context {
	ctrl.contextMutex.Lock()
	defer ctrl.contextMutex.Unlock()
	jobCtx, ok := ctrl.jobNodeContexts[jobID]
	if !ok {
		jobCtx, _ = system.Span(ctx, "controller",
			"JobLifecycle-"+ctrl.nodeID[:8],
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("jobID", jobID),
				attribute.String("nodeID", ctrl.nodeID),
			),
		)

		ctrl.jobNodeContexts[jobID] = jobCtx
	}
	return jobCtx
}

func (ctrl *Controller) addJobLifecycleEvent(ctx context.Context, jobID, eventName string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(eventName,
		trace.WithAttributes(
			append(attrs,
				attribute.String("jobID", jobID),
				attribute.String("nodeID", ctrl.nodeID),
			)...,
		),
	)
}

func (ctrl *Controller) newRootSpanForJob(ctx context.Context, jobID string) (context.Context, trace.Span) {
	jobCtx, span := system.Span(ctx, "controller", "JobLifecycle",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),

		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("jobID", jobID),
			attribute.String("nodeID", ctrl.nodeID),
		),
	)

	ctrl.jobContexts[jobID] = jobCtx

	return jobCtx, span
}
