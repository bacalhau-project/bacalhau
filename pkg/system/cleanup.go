package system

import (
	"context"
	"errors"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"

	realsync "sync"

	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/rs/zerolog/log"
)

const SleepBeforeCleanup = time.Millisecond * 100

// CleanupManager provides utilities for ensuring that sub-goroutines can
// clean up their resources before the main goroutine exits. Can be used to
// register callbacks for long-running system processes.
type CleanupManager struct {
	wg realsync.WaitGroup

	fnsMutex sync.Mutex
	fns      []func() error
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
	time.Sleep(SleepBeforeCleanup)

	// stop profiling now, just before we clean up, if we're profiling.
	log.Trace().Msg("============= STOPPING PROFILING ============")
	pprof.StopCPUProfile()
	memprofile := path.Join(os.TempDir(), "bacalhau-devstack-mem.prof")
	f, err := os.Create(memprofile)
	if err != nil {
		log.Info().Err(err).Msg("could not create memory profile")
	} else {
		defer closer.CloseWithLogOnError("mem.prof", f) // error handling omitted for example

		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Err(err).Msg("could not write memory profile")
		}
	}

	cm.fnsMutex.Lock()
	defer cm.fnsMutex.Unlock()

	if cm.fnsDone {
		log.Warn().Msg("CleanupManager: Cleanup called again after already called")
		return
	}

	for i := 0; i < len(cm.fns); i++ {
		go func(fn func() error) {
			defer cm.wg.Done()

			if err := fn(); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Error().Msgf("Error during clean-up callback: %v", err)
				}
			}
		}(cm.fns[i])
	}

	cm.wg.Wait()
	cm.fnsDone = true
}
