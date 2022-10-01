package job

import (
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

func ConstructJobFromEvent(ev model.JobEvent) *model.Job {
	log.Debug().Msgf("Constructing job from event: %+v", ev.JobID)
	log.Trace().Msgf("Full job event: %+v", ev)

	publicKey := ev.SenderPublicKey
	if publicKey == nil {
		publicKey = []byte{}
	}

	return &model.Job{
		APIVersion:         ev.APIVersion,
		ID:                 ev.JobID,
		RequesterNodeID:    ev.SourceNodeID,
		RequesterPublicKey: publicKey,
		ClientID:           ev.ClientID,
		Spec:               ev.Spec,
		Deal:               ev.Deal,
		ExecutionPlan:      ev.JobExecutionPlan,
		CreatedAt:          time.Now(),
	}
}

// these are util methods for the CLI
// to pass in the collection of CLI args as strings
// and have a Job struct returned
func ConstructDockerJob( //nolint:funlen
	a model.APIVersion,
	e model.Engine,
	v model.Verifier,
	p model.Publisher,
	cpu, memory, gpu string,
	inputUrls []string,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
	confidence int,
	minBids int,
	annotations []string,
	workingDir string,
	shardingGlobPattern string,
	shardingBasePath string,
	shardingBatchSize int,
	doNotTrack bool,
) (*model.Job, error) {
	if concurrency <= 0 {
		return &model.Job{}, fmt.Errorf("concurrency must be >= 1")
	}
	jobResources := model.ResourceUsageConfig{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}
	jobContexts := []model.StorageSpec{}

	jobInputs, err := buildJobInputs(inputVolumes, inputUrls)
	if err != nil {
		return &model.Job{}, err
	}
	jobOutputs, err := buildJobOutputs(outputVolumes)
	if err != nil {
		return &model.Job{}, err
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
		err = system.ValidateWorkingDir(workingDir)
		if err != nil {
			log.Error().Msg(err.Error())
			return &model.Job{}, err
		}
	}

	// Weird bug that sharding basepath fails if has a trailing slash
	shardingBasePath = strings.TrimSuffix(shardingBasePath, "/")

	jobShardingConfig := model.JobShardingConfig{
		GlobPattern: shardingGlobPattern,
		BasePath:    shardingBasePath,
		BatchSize:   shardingBatchSize,
	}

	j, err := model.NewJobWithSaneProductionDefaults()
	if err != nil {
		return &model.Job{}, err
	}
	j.APIVersion = a.String()

	j.Spec = model.Spec{
		Engine:    e,
		Verifier:  v,
		Publisher: p,
		Docker: model.JobSpecDocker{
			Image:                image,
			Entrypoint:           entrypoint,
			EnvironmentVariables: env,
		},

		Resources:   jobResources,
		Inputs:      jobInputs,
		Contexts:    jobContexts,
		Outputs:     jobOutputs,
		Annotations: jobAnnotations,
		Sharding:    jobShardingConfig,
		DoNotTrack:  doNotTrack,
	}

	// override working dir if provided
	if len(workingDir) > 0 {
		j.Spec.Docker.WorkingDirectory = workingDir
	}

	j.Deal = model.Deal{
		Concurrency: concurrency,
		Confidence:  confidence,
		MinBids:     minBids,
	}

	return j, nil
}

func ConstructLanguageJob(
	a model.APIVersion,
	inputVolumes []string,
	inputUrls []string,
	outputVolumes []string,
	env []string,
	concurrency int,
	confidence int,
	minBids int,
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
) (*model.Job, error) {
	// TODO refactor this wrt ConstructDockerJob
	if concurrency <= 0 {
		return &model.Job{}, fmt.Errorf("concurrency must be >= 1")
	}

	jobContexts := []model.StorageSpec{}

	jobInputs, err := buildJobInputs(inputVolumes, inputUrls)
	if err != nil {
		return &model.Job{}, err
	}
	jobOutputs, err := buildJobOutputs(outputVolumes)
	if err != nil {
		return &model.Job{}, err
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

	j, err := model.NewJobWithSaneProductionDefaults()
	if err != nil {
		return &model.Job{}, err
	}
	j.APIVersion = a.String()

	j.Spec = model.Spec{
		Engine:   model.EngineLanguage,
		Verifier: model.VerifierNoop,
		// TODO: should this always be ipfs?
		Publisher: model.PublisherIpfs,
		Language: model.JobSpecLanguage{
			Language:         language,
			LanguageVersion:  languageVersion,
			Deterministic:    deterministic,
			Context:          model.StorageSpec{},
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

	j.Deal = model.Deal{
		Concurrency: concurrency,
		Confidence:  confidence,
	}

	return j, err
}
