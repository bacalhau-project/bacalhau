package job

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// these are util methods for the CLI
// to pass in the collection of CLI args as strings
// and have a Job struct returned
func ConstructDockerJob( //nolint:funlen
	ctx context.Context,
	a model.APIVersion,
	e model.Engine,
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

	j.Spec = model.Spec{
		EngineSpec: (&model.JobSpecDocker{
			Image:                image,
			Entrypoint:           entrypoint,
			EnvironmentVariables: env,
			WorkingDirectory:     workingDir,
		}).AsEngineSpec(),
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
