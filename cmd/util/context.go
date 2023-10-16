package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type contextKey struct {
	name string
}

var (
	SystemManagerKey = contextKey{name: "context key for storing the system manager"}
	FSRepoKey        = contextKey{name: "context key for storing the filesystem-based repo"}
)

func GetCleanupManager(ctx context.Context) *system.CleanupManager {
	return ctx.Value(SystemManagerKey).(*system.CleanupManager)
}

func GetFSRepo(ctx context.Context) *repo.FsRepo {
	return ctx.Value(FSRepoKey).(*repo.FsRepo)
}
