package job

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

func ProcessJobIntoResults(job *executor.Job) (*[]types.ResultsList, error) {
	results := []types.ResultsList{}

	log.Debug().Msgf("All job states: %+v", job)

	log.Debug().Msgf("Number of job states created: %d", len(job.State))

	for node := range job.State {
		results = append(results, types.ResultsList{
			Node:   node,
			Cid:    job.State[node].ResultsId,
			Folder: system.GetResultsDirectory(job.Id, node),
		})
	}

	log.Debug().Msgf("Number of results created: %d", len(results))

	return &results, nil
}

func ConstructJob(
	engine executor.EngineType,
	verifier verifier.VerifierType,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
) (*executor.JobSpec, *executor.JobDeal, error) {
	if concurrency <= 0 {
		return nil, nil, fmt.Errorf("Concurrency must be >= 1")
	}

	jobInputs := []storage.StorageSpec{}
	jobOutputs := []storage.StorageSpec{}

	for _, inputVolume := range inputVolumes {
		slices := strings.Split(inputVolume, ":")
		if len(slices) != 2 {
			return nil, nil, fmt.Errorf("Invalid input volume: %s", inputVolume)
		}
		jobInputs = append(jobInputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Cid:    slices[0],
			Path:   slices[1],
		})
	}

	for _, outputVolume := range outputVolumes {
		slices := strings.Split(outputVolume, ":")
		if len(slices) != 2 {
			return nil, nil, fmt.Errorf("Invalid output volume: %s", outputVolume)
		}
		jobOutputs = append(jobOutputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Name:   slices[0],
			Path:   slices[1],
		})
	}

	spec := &executor.JobSpec{
		Engine:   engine,
		Verifier: verifier,
		Docker: executor.JobSpecDocker{
			Image:      image,
			Entrypoint: entrypoint,
			Env:        env,
		},

		Inputs:  jobInputs,
		Outputs: jobOutputs,
	}

	deal := &executor.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
}

func VerifyJob(spec *executor.JobSpec, Deal *executor.JobDeal) error {
	// TODO: do something useful here
	return nil
}
