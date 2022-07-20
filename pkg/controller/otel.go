package controller

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (ctrl *Controller) handleOtelReadEvent(ctx context.Context, ev executor.JobEvent) context.Context {

	return ctx
	//jobCtx := ctrl.getJobNodeContext(ctx, ev.JobID)
	//ctrl.addJobLifecycleEvent(jobCtx, ev.JobID, fmt.Sprintf("read_%s", ev.EventName))

	// fmt.Printf("ev.EventName --------------------------------------\n")
	// spew.Dump(ev.EventName.String())

	// // If the event is known to be ignorable, end the local lifecycle context:
	// if ev.EventName.IsIgnorable() {
	// 	fmt.Printf("IGNORE\n")
	// 	spew.Dump(ev.EventName.String())
	// 	ctrl.endJobNodeContext(ev.JobID)
	// }

	// // If the event is known to be terminal, end the global lifecycle context:
	// if ev.EventName.IsTerminal() {
	// 	fmt.Printf("TERMINAL\n")
	// 	spew.Dump(ev.EventName.String())
	// 	ctrl.endJobContext(ev.JobID)
	// }

	//return jobCtx
}

func (ctrl *Controller) cleanJobContexts(ctx context.Context) error {
	ctrl.contextMutex.RLock()
	defer ctrl.contextMutex.RUnlock()
	// End all job lifecycle spans so we don't lose any tracing data:
	for _, ctx := range ctrl.jobContexts {
		trace.SpanFromContext(ctx).End()
	}
	for _, ctx := range ctrl.jobNodeContexts {
		trace.SpanFromContext(ctx).End()
	}

	return nil
}

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
			"JobLifecycle-"+ctrl.id[:8],
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("jobID", jobID),
				attribute.String("nodeID", ctrl.id),
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
				attribute.String("nodeID", ctrl.id),
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
			attribute.String("nodeID", ctrl.id),
		),
	)

	ctrl.jobContexts[jobID] = jobCtx

	return jobCtx, span
}
