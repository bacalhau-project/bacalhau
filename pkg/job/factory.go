package job

import (
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
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
	workingDir string,
	doNotTrack bool,
) (executor.JobSpec, executor.JobDeal, error) {
	if concurrency <= 0 {
		return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("concurrency must be >= 1")
	}
	jobResources := capacitymanager.ResourceUsageConfig{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	jobContexts := []storage.StorageSpec{}

	jobInputs, err := buildJobInputs(inputVolumes, inputUrls)
	if err != nil {
		return executor.JobSpec{}, executor.JobDeal{}, err
	}
	jobOutputs, err := buildJobOutputs(outputVolumes)
	if err != nil {
		return executor.JobSpec{}, executor.JobDeal{}, err
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
		Contexts:    jobContexts,
		Outputs:     jobOutputs,
		Annotations: jobAnnotations,
		DoNotTrack:  doNotTrack,
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
	inputUrls []string,
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
	annotations []string,
	doNotTrack bool,
) (executor.JobSpec, executor.JobDeal, error) {
	// TODO refactor this wrt ConstructDockerJob

	if concurrency <= 0 {
		return executor.JobSpec{}, executor.JobDeal{}, fmt.Errorf("concurrency must be >= 1")
	}

	jobContexts := []storage.StorageSpec{}

	jobInputs, err := buildJobInputs(inputVolumes, inputUrls)
	if err != nil {
		return executor.JobSpec{}, executor.JobDeal{}, err
	}
	jobOutputs, err := buildJobOutputs(outputVolumes)
	if err != nil {
		return executor.JobSpec{}, executor.JobDeal{}, err
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
		Inputs:      jobInputs,
		Contexts:    jobContexts,
		Outputs:     jobOutputs,
		Annotations: jobAnnotations,
		DoNotTrack:  doNotTrack,
	}

	deal := executor.JobDeal{
		Concurrency: concurrency,
	}

	return spec, deal, nil
}
