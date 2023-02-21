package telemetry

import (
	"context"
	"time"
)

// NewDetachedContext produces a new context that has a separate cancellation mechanism from its parent. This should be
// used when having to continue using a context after it has been canceled, such as cleaning up Docker resources
// after the context has been canceled but want to keep work associated with the same trace.
func NewDetachedContext(parent context.Context) context.Context {
	return detachedContext{parent: parent}
}

var _ context.Context = detachedContext{}

type detachedContext struct {
	parent context.Context
}

func (d detachedContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (d detachedContext) Done() <-chan struct{} {
	return nil
}

func (d detachedContext) Err() error {
	return nil
}

func (d detachedContext) Value(key any) any {
	return d.parent.Value(key)
}
