package job

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

func ConstructDockerJob( //nolint:funlen
	engine executor.EngineType,
	v verifier.VerifierType,
	cpu, memory, gpu string,
	inputUrls []string,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
	annotations []string,
	workingDir string,
) (executor.JobSpec, executor.JobDeal, error) {
	if concurrency <= 0 {
		return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("concurrency must be >= 1")
	}
	jobResources := capacitymanager.ResourceUsageConfig{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	jobInputs := []storage.StorageSpec{}
	jobOutputs := []storage.StorageSpec{}

	for _, inputURL := range inputUrls {
		// split using LastIndex to support port numbers in URL
		lastInd := strings.LastIndex(inputURL, ":")
		rawURL := inputURL[:lastInd]
		path := inputURL[lastInd+1:]
		// should loop through all available storage providers?
		_, err := urldownload.IsURLSupported(rawURL)
		if err != nil {
			return executor.JobSpec{}, executor.JobDeal{}, err
		}
		jobInputs = append(jobInputs, storage.StorageSpec{
			Engine: storage.StorageSourceURLDownload,
			URL:    rawURL,
			Path:   path,
		})
	}

	for _, inputVolume := range inputVolumes {
		slices := strings.Split(inputVolume, ":")
		if len(slices) != 2 {
			return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("invalid input volume: %s", inputVolume)
		}
		if strings.Contains(slices[0], "/") {
			return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("forward slash in CID not (yet) supported: %s", slices[0])
		}
		jobInputs = append(jobInputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: storage.StorageSourceIPFS,
			Cid:    slices[0],
			Path:   slices[1],
		})
	}

	for _, outputVolume := range outputVolumes {
		slices := strings.Split(outputVolume, ":")
		if len(slices) != 2 {
			msg := fmt.Sprintf("invalid output volume: %s", outputVolume)
			log.Error().Msgf(msg)
			return executor.JobSpec{}, executor.JobDeal{}, errors.New(msg)
		}
		jobOutputs = append(jobOutputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: storage.StorageSourceIPFS,
			Name:   slices[0],
			Path:   slices[1],
		})
	}

	var jobAnnotations []string
	var unSafeAnnotations []string
	for _, a := range annotations {
		if IsSafeAnnotation(a) && a != "" {
			jobAnnotations = append(jobAnnotations, a)
		} else {
			unSafeAnnotations = append(unSafeAnnotations, a)
		}
	}

	if len(unSafeAnnotations) > 0 {
		log.Error().Msgf("The following labels are unsafe. Labels must fit the regex '/%s/' (and all emjois): %+v",
			RegexString,
			strings.Join(unSafeAnnotations, ", "))
	}

	if len(workingDir) > 0 {
		err := system.ValidateWorkingDir(workingDir)
		if err != nil {
			log.Error().Msg(err.Error())
			return executor.JobSpec{}, executor.JobDeal{}, err
		}
	}

	spec := executor.JobSpec{
		Engine:   engine,
		Verifier: v,
		Docker: executor.JobSpecDocker{
			Image:      image,
			Entrypoint: entrypoint,
			Env:        env,
		},

		Resources:   jobResources,
		Inputs:      jobInputs,
		Outputs:     jobOutputs,
		Annotations: jobAnnotations,
	}

	// override working dir if provided
	if len(workingDir) > 0 {
		spec.Docker.WorkingDir = workingDir
	}

	deal := executor.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
}

func ConstructLanguageJob(
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	concurrency int,
	// See JobSpecLanguage
	language string,
	languageVersion string,
	command string,
	programPath string,
	requirementsPath string,
	contextPath string, // we have to tar this up and POST it to the requestor node
	deterministic bool,
) (executor.JobSpec, executor.JobDeal, error) {
	// TODO refactor this wrt ConstructDockerJob

	if concurrency <= 0 {
		return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("concurrency must be >= 1")
	}

	jobInputs := []storage.StorageSpec{}
	jobOutputs := []storage.StorageSpec{}

	for _, inputVolume := range inputVolumes {
		slices := strings.Split(inputVolume, ":")
		if len(slices) != 2 {
			return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("invalid input volume: %s", inputVolume)
		}
		jobInputs = append(jobInputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: storage.StorageSourceIPFS,
			Cid:    slices[0],
			Path:   slices[1],
		})
	}

	for _, outputVolume := range outputVolumes {
		slices := strings.Split(outputVolume, ":")
		if len(slices) != 2 {
			return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("invalid output volume: %s", outputVolume)
		}
		jobOutputs = append(jobOutputs, storage.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: storage.StorageSourceIPFS,
			Name:   slices[0],
			Path:   slices[1],
		})
	}

	spec := executor.JobSpec{
		Engine: executor.EngineLanguage,
		// TODO: should this always be ipfs?
		Verifier: verifier.VerifierIpfs,
		Language: executor.JobSpecLanguage{
			Language:         language,
			LanguageVersion:  languageVersion,
			Deterministic:    deterministic,
			Context:          storage.StorageSpec{},
			Command:          command,
			ProgramPath:      programPath,
			RequirementsPath: requirementsPath,
		},

		Inputs:  jobInputs,
		Outputs: jobOutputs,
	}

	deal := executor.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
}

func VerifyJob(spec executor.JobSpec, deal executor.JobDeal) error {
	if reflect.DeepEqual(executor.JobSpec{}, spec) {
		return fmt.Errorf("job spec is empty")
	}

	if reflect.DeepEqual(executor.JobDeal{}, deal) {
		return fmt.Errorf("job spec is empty")
	}

	return nil
}

// TODO: #259 We need to rename this - what does it mean to be "furthest along" for a job? Closest to final?
func GetCurrentJobState(states map[string]executor.JobState) (string, executor.JobState) {
	// Returns Node Id, JobState

	// Combine the list of jobs down to just those that matter
	// Strategy here is assuming the following:
	// - All created times are the same (we'll choose the biggest, but it shouldn't matter)
	// - All Job IDs are the same (we'll use it as the anchor to combine)
	// - If a job has all "bid_rejected", then that's the answer for state
	// - If a job has anything BUT bid rejected, then that's the answer for state
	// - Everything else SHOULD be equivalent, but doesn't matter (really), so we'll just show the
	// 	 one that has the non-bid-rejected result.

	finalNodeID := ""
	finalJobState := executor.JobState{}

	for nodeID, jobState := range states {
		if finalNodeID == "" {
			finalNodeID = nodeID
			finalJobState = jobState
		} else if JobStateValue(jobState) > JobStateValue(finalJobState) {
			// Overwrite any states that are there with a new state - so we only have one
			finalNodeID = nodeID
			finalJobState = jobState
		}
	}
	return finalNodeID, finalJobState
}

func JobStateValue(jobState executor.JobState) int {
	return int(jobState.State)
}
