package job

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

func ConstructJobFromEvent(ev executor.JobEvent) executor.Job {
	log.Debug().Msgf("Constructing job from event: %+v", ev)
	return executor.Job{
		ID:              ev.JobID,
		RequesterNodeID: ev.SourceNodeID,
		ClientID:        ev.ClientID,
		Spec:            ev.JobSpec,
		Deal:            ev.JobDeal,
		ExecutionPlan:   ev.JobExecutionPlan,
		CreatedAt:       time.Now(),
	}
}

// these are util methods for the CLI
// to pass in the collection of CLI args as strings
// and have a Job struct returned
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
	jobContexts := []storage.StorageSpec{}
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
		Contexts:    jobContexts,
		Outputs:     jobOutputs,
		Annotations: jobAnnotations,
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
	jobContexts := []storage.StorageSpec{}
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

		Inputs:   jobInputs,
		Contexts: jobContexts,
		Outputs:  jobOutputs,
	}

	deal := executor.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
}
