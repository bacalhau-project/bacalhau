package boltdb

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsOperation(slice []watcher.Operation, item watcher.Operation) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
