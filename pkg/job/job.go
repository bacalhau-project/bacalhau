package job

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

func ProcessJobIntoResults(job *types.Job) (*[]types.ResultsList, error) {
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
	engine string,
	verifier string,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
	labels []string,
) (*types.JobSpec, *types.JobDeal, error) {
	if concurrency <= 0 {
		return nil, nil, fmt.Errorf("Concurrency must be >= 1")
	}

	jobInputs := []types.StorageSpec{}
	jobOutputs := []types.StorageSpec{}

	for _, inputVolume := range inputVolumes {
		slices := strings.Split(inputVolume, ":")
		if len(slices) != 2 {
			return nil, nil, fmt.Errorf("Invalid input volume: %s", inputVolume)
		}
		jobInputs = append(jobInputs, types.StorageSpec{
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
		jobOutputs = append(jobOutputs, types.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: "ipfs",
			Name:   slices[0],
			Path:   slices[1],
		})
	}

	var jobLabels []string

	for _, label := range labels {
		if label != "" {
			jobLabels = append(jobLabels, label)
		}
	}

	spec := &types.JobSpec{
		Engine:   engine,
		Verifier: verifier,
		Vm: types.JobSpecVm{
			Image:      image,
			Entrypoint: entrypoint,
			Env:        env,
		},

		Inputs:  jobInputs,
		Outputs: jobOutputs,
		Labels:  jobLabels,
	}

	deal := &types.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
}

func VerifyJob(spec *types.JobSpec, Deal *types.JobDeal) error {
	if !system.StringArrayContains(executor.EXECUTORS, spec.Engine) {
		return fmt.Errorf("Invalid executor: %s", spec.Engine)
	}
	if !system.StringArrayContains(verifier.VERIFIERS, spec.Verifier) {
		return fmt.Errorf("Invalid verifier: %s", spec.Verifier)
	}
	return nil
}

// TODO: #259 We need to rename this - what does it mean to be "furthest along" for a job? Closest to final?
func GetCurrentJobState(job *types.Job) (string, *types.JobState) {
	// Combine the list of jobs down to just those that matter
	// Strategy here is assuming the following:
	// - All created times are the same (we'll choose the biggest, but it shouldn't matter)
	// - All Job IDs are the same (we'll use it as the anchor to combine)
	// - If a job has all "bid_rejected", then that's the answer for state
	// - If a job has anything BUT bid rejected, then that's the answer for state
	// - Everything else SHOULD be equivalent, but doesn't matter (really), so we'll just show the
	// 	 one that has the non-bid-rejected result.

	finalNodeId := ""
	finalJobState := &types.JobState{}

	for nodeId, jobState := range job.State {
		if finalNodeId == "" {
			finalNodeId = nodeId
			finalJobState = jobState
		} else if JobStateValue(jobState) > JobStateValue(finalJobState) {
			// Overwrite any states that are there with a new state - so we only have one
			finalNodeId = nodeId
			finalJobState = jobState
		}
	}
	return finalNodeId, finalJobState
}

func JobStateValue(jobState *types.JobState) int {
	switch jobState.State {
	case types.JOB_STATE_RUNNING:
		return 100
	case types.JOB_STATE_COMPLETE:
		return 90
	case types.JOB_STATE_ERROR:
		return 80
	case types.JOB_STATE_BIDDING:
		return 70
	case types.JOB_STATE_BID_REJECTED:
		return 60
	default:
		log.Error().Msgf("Asking value with unknown state. State: %+v", jobState.State)
		return 0
	}
}
