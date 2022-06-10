package system

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// CleanupManager provides utilities for ensuring that sub-goroutines can
// clean up their resources before the main goroutine exits. Can be used to
// register callbacks for long-running system processes.
type CleanupManager struct {
	wg  sync.WaitGroup
	fns []func() error
}

// NewCleanupManager returns a new CleanupManager instance.
func NewCleanupManager() *CleanupManager {
	return &CleanupManager{}
}

// RegisterCallback registers a clean-up function.
func (cm *CleanupManager) RegisterCallback(fn func() error) {
	cm.wg.Add(1)
	cm.fns = append(cm.fns, fn)
}

// Cleanup runs all registered clean-up functions in sub-goroutines and
// waits for them all to complete before exiting.
func (cm *CleanupManager) Cleanup() {
	for i := 0; i < len(cm.fns); i++ {
		go func(fn func() error) {
			defer cm.wg.Done()

			if err := fn(); err != nil {
				log.Error().Msgf("Error during clean-up callback: %v", err)
			}
		}(cm.fns[i])
	}

	cm.wg.Wait()
}
