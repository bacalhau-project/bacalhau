package job

import (
	"context"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// these are util methods for the CLI
// to pass in the collection of CLI args as strings
// and have a Job struct returned
func ConstructDockerJob( //nolint:funlen
	ctx context.Context,
	a model.APIVersion,
	v model.Verifier,
	p model.PublisherSpec,
	cpu, memory, gpu string,
	network model.Network,
	domains []string,
	inputs []model.StorageSpec,
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
) (*model.Job, error) {
	jobResources := model.ResourceUsageConfig{
		CPU:    cpu,
		Memory: memory,
		GPU:    gpu,
	}

	jobOutputs, err := BuildJobOutputs(ctx, outputVolumes)
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
		log.Ctx(ctx).Error().Msgf("The following labels are unsafe. Labels must fit the regex '/%s/' (and all emjois): %+v",
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
			return &model.Job{}, err
		}
	}

	j, err := model.NewJobWithSaneProductionDefaults()
	if err != nil {
		return &model.Job{}, err
	}
	j.APIVersion = a.String()

	dockerEngine := docker.DockerEngineSpec{
		Image:                image,
		Entrypoint:           entrypoint,
		EnvironmentVariables: env,
	}
	// override working dir if provided
	if len(workingDir) > 0 {
		dockerEngine.WorkingDirectory = workingDir
	}
	engine, err := dockerEngine.AsSpec()
	if err != nil {
		return nil, err
	}
	j.Spec = model.Spec{
		Engine:        engine,
		Verifier:      v,
		PublisherSpec: p,
		Network: model.NetworkConfig{
			Type:    network,
			Domains: domains,
		},
		Timeout:       timeout,
		Resources:     jobResources,
		Inputs:        inputs,
		Outputs:       jobOutputs,
		Annotations:   jobAnnotations,
		NodeSelectors: nodeSelectorRequirements,
	}

	j.Spec.Deal = model.Deal{
		Concurrency: concurrency,
		Confidence:  confidence,
		MinBids:     minBids,
	}

	return j, nil
}

func ConstructLanguageJob(
	ctx context.Context,
	inputs []model.StorageSpec,
	outputVolumes []string,
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
	deterministic bool,
	annotations []string,
) (*model.Job, error) {
	// TODO refactor this wrt ConstructDockerJob

	jobOutputs, err := buildJobOutputs(ctx, outputVolumes)
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
		log.Ctx(ctx).Error().Msgf("The following labels are unsafe. Labels must fit the regex '/%s/' (and all emjois): %+v",
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
	j.Spec.Inputs = inputs
	j.Spec.Outputs = jobOutputs
	j.Spec.Annotations = jobAnnotations

	j.Spec.Deal = model.Deal{
		Concurrency: concurrency,
		Confidence:  confidence,
		MinBids:     minBids,
	}

	return j, err
}
