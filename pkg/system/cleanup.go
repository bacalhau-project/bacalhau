package system

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CleanupManager provides utilities for ensuring that sub-goroutines can
// clean up their resources before the main goroutine exits. Can be used to
// register callbacks for long-running system processes.
type CleanupManager struct {
	wg sync.WaitGroup

	fnsMutex sync.Mutex
	fns      []func() error
	fnsDone  bool
}

// NewCleanupManager returns a new CleanupManager instance.
func NewCleanupManager() *CleanupManager {
	return &CleanupManager{}
}

// RegisterCallback registers a clean-up function.
func (cm *CleanupManager) RegisterCallback(fn func() error) {
	cm.fnsMutex.Lock()
	defer cm.fnsMutex.Unlock()

	if cm.fnsDone {
		log.Error().Msg("CleanupManager: RegisterCallback called after Cleanup")
		return
	}

	cm.wg.Add(1)
	cm.fns = append(cm.fns, fn)
}

// Cleanup runs all registered clean-up functions in sub-goroutines and
// waits for them all to complete before exiting.
func (cm *CleanupManager) Cleanup() {
	// we sleep a tiny bit here because some tests run so quickly
	// that there are RegisterCallback calls happening
	// after we have been called
	time.Sleep(time.Millisecond * 100)

	cm.fnsMutex.Lock()
	defer cm.fnsMutex.Unlock()

	for i := 0; i < len(cm.fns); i++ {
		go func(fn func() error) {
			defer cm.wg.Done()

			if err := fn(); err != nil {
				if err.Error() != "context canceled" {
					log.Error().Msgf("Error during clean-up callback: %v", err)
				}
			}
		}(cm.fns[i])
	}

	cm.wg.Wait()
	cm.fnsDone = true
}
