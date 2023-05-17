package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

type Environment struct {
	Execution       store.Execution
	InputVolumes    map[*model.StorageSpec]storage.StorageVolume
	OutputFolder    string
	OutputVolumes   map[*model.StorageSpec]storage.StorageVolume
	StorageProvider storage.StorageProvider
}

func NewEnvironment(execution store.Execution, storageProvider storage.StorageProvider) (*Environment, error) {
	return &Environment{
		Execution:       execution,
		InputVolumes:    make(map[*model.StorageSpec]storage.StorageVolume),
		OutputVolumes:   make(map[*model.StorageSpec]storage.StorageVolume),
		StorageProvider: storageProvider,
	}, nil
}

func (e *Environment) Build(
	ctx context.Context,
	verifier verifier.Verifier) error {
	// Ensure output path is created
	outputFolder, err := verifier.GetResultPath(ctx, e.Execution.ID, e.Execution.Job)
	if err != nil {
		return err
	}
	e.OutputFolder = outputFolder

	// Noop won't have a storage provider and so we shouldn't try and download
	// anything.  It tells us this with a nil pointer :(
	// TODO: We need to understand what happens if the nop executor has a handler
	// installed.
	if e.StorageProvider == nil {
		return nil
	}

	// For specific outputs in spec, make sure the directory exists under output
	// folder ready for mounting by the executor
	for i := range e.Execution.Job.Spec.Outputs {
		output := e.Execution.Job.Spec.Outputs[i]

		if output.Name == "" {
			err = fmt.Errorf("output volume has no name: %+v", output)
			return err
		}

		if output.Path == "" {
			err = fmt.Errorf("output volume has no path: %+v", output)
			return err
		}

		srcd := filepath.Join(e.OutputFolder, output.Name)
		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return err
		}

		e.OutputVolumes[&output] = storage.StorageVolume{
			Type:   storage.StorageVolumeConnectorBind,
			Source: srcd,
			Target: output.Path,
		}
	}

	// Process inputs
	e.InputVolumes, err = storage.ParallelPrepareStorage(ctx, e.StorageProvider, e.Execution.Job.Spec.Inputs)
	if err != nil {
		return err
	}

	return nil
}

func (e *Environment) Destroy(ctx context.Context) error {
	// Can clean up inputs, but not allowed to clean up output as it will be used
	// later by Publish which will take responsibility for publishing the output
	for specification, volume := range e.InputVolumes {
		spec := specification

		// we can tell it to cleanup the storage at the end of the execution.
		provider, err := e.StorageProvider.Get(ctx, spec.StorageSource)
		if err != nil {
			log.Ctx(ctx).Error().
				Err(err).
				Str("Source", spec.StorageSource.String()).
				Msg("failed to get storage provider in cleanup")
			return err
		}

		log.Ctx(ctx).Debug().
			Str("Execution", e.Execution.ID).
			Msg("cleaning up inputs for execution")

		// May be noop depending on the storage provider
		err = provider.CleanupStorage(ctx, *spec, volume)
		if err != nil {
			log.Ctx(ctx).Error().
				Err(err).
				Str("Source", spec.StorageSource.String()).
				Msg("failed to cleanup volume")
		}
	}

	return nil
}
