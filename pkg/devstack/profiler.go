package devstack

import (
	"context"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
)

type profiler struct {
	cpuFile    *os.File
	memoryFile string
}

func StartProfiling(ctx context.Context, cpuFile, memoryFile string) CloserWithContext {
	// do a GC before we start profiling
	runtime.GC()

	log.Ctx(ctx).Trace().Msg("============= STARTING PROFILING ============")

	var f *os.File
	if cpuFile != "" {
		var err error
		f, err = os.Create(cpuFile)
		if err != nil {
			log.Ctx(ctx).Debug().Err(err).Str("Path", cpuFile).Msg("could not create CPU profile")
			return nil
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			closer.CloseWithLogOnError(cpuFile, f)
			log.Ctx(ctx).Debug().Err(err).Msg("could not start CPU profile")
			return nil
		}
	}

	return &profiler{cpuFile: f, memoryFile: memoryFile}
}

func (p *profiler) Close(ctx context.Context) error {
	// stop profiling now, just before we clean up, if we're profiling.
	log.Trace().Msg("============= STOPPING PROFILING ============")
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		closer.CloseWithLogOnError(p.cpuFile.Name(), p.cpuFile)
	}

	if p.memoryFile != "" {
		f, err := os.Create(p.memoryFile)
		if err != nil {
			log.Ctx(ctx).Debug().Err(err).Str("Path", p.memoryFile).Msg("could not create memory profile")
			return nil
		}
		defer closer.CloseWithLogOnError(p.memoryFile, f) // error handling omitted for example

		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Ctx(ctx).Debug().Err(err).Msg("could not write memory profile")
		}
	}

	return nil
}

var _ CloserWithContext = (*profiler)(nil)

type CloserWithContext interface {
	// Close closes the resource.
	Close(context.Context) error
}
