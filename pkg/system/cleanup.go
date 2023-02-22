package system

import (
	"context"
	"errors"
	realsync "sync"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/pkg/telemetry"
	"github.com/rs/zerolog/log"
)

// CleanupManager provides utilities for ensuring that sub-goroutines can
// clean up their resources before the main goroutine exits. Can be used to
// register callbacks for long-running system processes.
type CleanupManager struct {
	fnsMutex sync.Mutex
	fns      []any
	fnsDone  bool
}

// NewCleanupManager returns a new CleanupManager instance.
func NewCleanupManager() *CleanupManager {
	c := &CleanupManager{}
	c.fnsMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "CleanupManager.fnsMutex",
	})
	return c
}

// RegisterCallback registers a clean-up function.
func (cm *CleanupManager) RegisterCallback(fn cleanUpWithoutContext) {
	cm.registerCallback(fn)
}

// RegisterCallbackWithContext registers a clean-up function. The context passed is guaranteed not to be already canceled.
func (cm *CleanupManager) RegisterCallbackWithContext(fn cleanUpWithContext) {
	cm.registerCallback(fn)
}

func (cm *CleanupManager) registerCallback(fn any) {
	cm.fnsMutex.Lock()
	defer cm.fnsMutex.Unlock()

	if cm.fnsDone {
		log.Error().Msg("CleanupManager: RegisterCallback called after Cleanup")
		return
	}

	cm.fns = append(cm.fns, fn)
}

// Cleanup runs all registered clean-up functions in sub-goroutines and
// waits for them all to complete before exiting.
func (cm *CleanupManager) Cleanup(ctx context.Context) {
	cm.fnsMutex.Lock()
	defer cm.fnsMutex.Unlock()

	if cm.fnsDone {
		log.Ctx(ctx).Warn().Msg("CleanupManager: Cleanup called again after already called")
		return
	}

	var wg realsync.WaitGroup
	wg.Add(len(cm.fns))

	detachedContext := telemetry.NewDetachedContext(ctx)

	for i := 0; i < len(cm.fns); i++ {
		go func(fn any) {
			defer wg.Done()

			var err error
			switch f := fn.(type) {
			case cleanUpWithContext:
				err = f(detachedContext)
			case cleanUpWithoutContext:
				err = f()
			}

			if err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Ctx(detachedContext).Error().Err(err).Msg("Error during clean-up callback")
				}
			}
		}(cm.fns[i])
	}

	wg.Wait()
	cm.fnsDone = true
}

type cleanUpWithoutContext func() error
type cleanUpWithContext func(context.Context) error
