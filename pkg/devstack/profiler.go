package devstack

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
)

const (
	cpuProfile string = "bacalhau-devstack-cpu.prof"
	memProfile string = "bacalhau-devstack-mem.prof"
)

type profiler struct {
	cpuFile *os.File
}

func StartProfiling() io.Closer {
	// do a GC before we start profiling
	runtime.GC()

	log.Trace().Msg("============= STARTING PROFILING ============")
	// devstack always records a cpu profile, it will be generally useful.
	cpuprofile := filepath.Join(os.TempDir(), cpuProfile)
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Debug().Err(err).Str("Path", cpuprofile).Msg("could not create CPU profile")
		return nil
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		closer.CloseWithLogOnError(cpuprofile, f)
		log.Debug().Err(err).Msg("could not start CPU profile")
		return nil
	}

	closer := profiler{cpuFile: f}
	return &closer
}

func (p *profiler) Close() error {
	// stop profiling now, just before we clean up, if we're profiling.
	log.Trace().Msg("============= STOPPING PROFILING ============")
	pprof.StopCPUProfile()
	closer.CloseWithLogOnError(p.cpuFile.Name(), p.cpuFile)

	memprofile := filepath.Join(os.TempDir(), memProfile)
	f, err := os.Create(memprofile)
	if err != nil {
		log.Debug().Err(err).Str("Path", memprofile).Msg("could not create memory profile")
		return nil
	}
	defer closer.CloseWithLogOnError(memprofile, f) // error handling omitted for example

	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Debug().Err(err).Msg("could not write memory profile")
	}
	return nil
}

var _ io.Closer = (*profiler)(nil)
