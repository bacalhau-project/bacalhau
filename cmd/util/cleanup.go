package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

type contextKey struct {
	name string
}

var SystemManagerKey = contextKey{name: "context key for storing the system manager"}

func GetCleanupManager(ctx context.Context) *system.CleanupManager {
	return ctx.Value(SystemManagerKey).(*system.CleanupManager)
}

// injects a Cleanup Manager into the context.
// TODO deprecated this and the CleanupManager.
func InjectCleanupManager(ctx context.Context) context.Context {
	cm := system.NewCleanupManager()
	cm.RegisterCallback(telemetry.Cleanup)
	return context.WithValue(ctx, SystemManagerKey, cm)
}
