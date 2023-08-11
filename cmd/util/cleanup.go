package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type contextKey struct {
	name string
}

var SystemManagerKey = contextKey{name: "context key for storing the system manager"}

func GetCleanupManager(ctx context.Context) *system.CleanupManager {
	return ctx.Value(SystemManagerKey).(*system.CleanupManager)
}
