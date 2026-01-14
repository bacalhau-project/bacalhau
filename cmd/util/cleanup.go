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

// Profile context keys for --profile flag and BACALHAU_PROFILE env var
var ProfileFlagKey = contextKey{name: "profile flag"}
var ProfileEnvKey = contextKey{name: "profile env"}

// GetProfileFromContext retrieves profile selection from command context.
// Returns the flag value and env value.
func GetProfileFromContext(ctx context.Context) (flagValue, envValue string) {
	if v := ctx.Value(ProfileFlagKey); v != nil {
		flagValue = v.(string)
	}
	if v := ctx.Value(ProfileEnvKey); v != nil {
		envValue = v.(string)
	}
	return
}

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
