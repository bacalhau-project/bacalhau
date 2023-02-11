package job

import (
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// these are util methods for the CLI
// to pass in the collection of CLI args as strings
// and have a Job struct returned
func ConstructDockerJob( //nolint:funlen
	a model.APIVersion,
	e model.Engine,
	v model.Verifier,
	p model.Publisher,
	cpu, memory, gpu string,
	network model.Network,
	domains []string,
	inputUrls []string,
	inputVolumes []string,
	outputVolumes []string,
	env []string,
	entrypoint []string,
	image string,
	concurrency int,
	confidence int,
	minBids int,
	timeout float64,
	annotations []string,
	nodeSelector string,
	workingDir string,
	shardingGlobPattern string,
	shardingBasePath string,
	shardingBatchSize int,
	doNotTrack bool,
) (*model.Job, error) {
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

	nodeSelectorRequirements, err := ParseNodeSelector(nodeSelector)
	if err != nil {
		return &model.Job{}, err
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
		Network: model.NetworkConfig{
			Type:    network,
			Domains: domains,
		},
		Timeout:       timeout,
		Resources:     jobResources,
		Inputs:        jobInputs,
		Contexts:      jobContexts,
		Outputs:       jobOutputs,
		Annotations:   jobAnnotations,
		NodeSelectors: nodeSelectorRequirements,
		Sharding:      jobShardingConfig,
		DoNotTrack:    doNotTrack,
	}

	// override working dir if provided
	if len(workingDir) > 0 {
		j.Spec.Docker.WorkingDirectory = workingDir
	}

	j.Spec.Deal = model.Deal{
		Concurrency: concurrency,
		Confidence:  confidence,
		MinBids:     minBids,
	}

	return j, nil
}

func ConstructLanguageJob(
	inputVolumes []string,
	inputUrls []string,
	outputVolumes []string,
	env []string,
	concurrency int,
	confidence int,
	minBids int,
	timeout float64,
	// See JobSpecLanguage
	language string,
	languageVersion string,
	command string,
	programPath string,
	requirementsPath string,
	contextPath string, // we have to tar this up and POST it to the Requester node
	deterministic bool,
	annotations []string,
	doNotTrack bool,
) (*model.Job, error) {
	// TODO refactor this wrt ConstructDockerJob
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

	j.Spec.Engine = model.EngineLanguage
	j.Spec.Language = model.JobSpecLanguage{
		Language:         language,
		LanguageVersion:  languageVersion,
		Deterministic:    deterministic,
		Context:          model.StorageSpec{},
		Command:          command,
		ProgramPath:      programPath,
		RequirementsPath: requirementsPath,
	}
	j.Spec.Timeout = timeout
	j.Spec.Inputs = jobInputs
	j.Spec.Contexts = jobContexts
	j.Spec.Outputs = jobOutputs
	j.Spec.Annotations = jobAnnotations
	j.Spec.DoNotTrack = doNotTrack

	j.Spec.Deal = model.Deal{
		Concurrency: concurrency,
		Confidence:  confidence,
		MinBids:     minBids,
	}

	return j, err
}
