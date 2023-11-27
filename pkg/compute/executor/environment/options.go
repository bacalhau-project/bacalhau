package environment

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type Option func(e *Environment)

func WithStorageProvider(prov *provider.Provider[storage.Storage]) Option {
	return func(e *Environment) {
		e.storage = prov
	}
}

func WithCleanupPolicy(policy *CleanupPolicy) Option {
	return func(e *Environment) {
		e.cleanupPolicy = policy
	}
}
