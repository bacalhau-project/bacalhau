package job

import (
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

<<<<<<< HEAD
type SpecOpt func(s *model.Spec) error
=======
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
	entrypoint string,
	cmd []string,
	image string,
	deal model.Deal,
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
>>>>>>> fbbf47f7 (Change entrypoint to be a string and other things...)

func WithVerifier(v model.Verifier) SpecOpt {
	return func(s *model.Spec) error {
		s.Verifier = v
		return nil
	}
}

func WithPublisher(p model.PublisherSpec) SpecOpt {
	return func(s *model.Spec) error {
		s.Publisher = p.Type
		s.PublisherSpec = p
		return nil
	}
}

func WithNetwork(network model.Network, domains []string) SpecOpt {
	return func(s *model.Spec) error {
		s.Network.Type = network
		s.Network.Domains = domains
		return nil
	}
}

func WithResources(cpu, memory, disk, gpu string) SpecOpt {
	return func(s *model.Spec) error {
		s.Resources.CPU = cpu
		s.Resources.Memory = memory
		s.Resources.Disk = disk
		s.Resources.GPU = gpu
		return nil
	}
}

func WithTimeout(t float64) SpecOpt {
	return func(s *model.Spec) error {
		s.Timeout = t
		return nil
	}
}

func WithDeal(targeting model.TargetingMode, concurrency, confidence, minbids int) SpecOpt {
	return func(s *model.Spec) error {
		s.Deal.TargetingMode = targeting
		s.Deal.Concurrency = concurrency
		s.Deal.Confidence = confidence
		s.Deal.MinBids = minbids
		return nil
	}
}

func WithAnnotations(annotations ...string) SpecOpt {
	return func(s *model.Spec) error {
		s.Annotations = annotations
		return nil
	}
}

func WithInputs(inputs ...model.StorageSpec) SpecOpt {
	return func(s *model.Spec) error {
		s.Inputs = inputs
		return nil
	}
}

func WithOutputs(outputs ...model.StorageSpec) SpecOpt {
	return func(s *model.Spec) error {
		s.Outputs = outputs
		return nil
	}
}

func WithNodeSelector(selector []model.LabelSelectorRequirement) SpecOpt {
	return func(s *model.Spec) error {
		s.NodeSelectors = selector
		return nil
	}
}

func WithDockerEngine(image, workdir string, entrypoint, envvar, parameters []string) SpecOpt {
	return func(s *model.Spec) error {
		if err := system.ValidateWorkingDir(workdir); err != nil {
			return fmt.Errorf("validating docker working directory: %w", err)
		}
<<<<<<< HEAD
		s.Engine = model.EngineDocker
		s.Docker = model.JobSpecDocker{
			Image:                image,
			Entrypoint:           entrypoint,
			Parameters:           parameters,
			EnvironmentVariables: envvar,
			WorkingDirectory:     workdir,
		}
		return nil
=======
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
	var entrypointSlice []string
	if entrypoint != "" {
		entrypointSlice = []string{entrypoint}
	}
	j.Spec = model.Spec{
		Engine:        e,
		Verifier:      v,
		PublisherSpec: p,
		Docker: model.JobSpecDocker{
			Image:                image,
			Entrypoint:           entrypointSlice,
			EnvironmentVariables: env,
			Parameters:           cmd,
		},
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
>>>>>>> fbbf47f7 (Change entrypoint to be a string and other things...)
	}
}

func MakeDockerSpec(
	image, workingdir string, entrypoint, envvar, parameters []string,
	opts ...SpecOpt,
) (model.Spec, error) {
	spec, err := MakeSpec(append(opts, WithDockerEngine(image, workingdir, entrypoint, envvar, parameters))...)
	if err != nil {
		return model.Spec{}, err
	}
	return spec, nil
}

const null rune = 0

func WithWasmEngine(
	entryModule model.StorageSpec,
	entrypoint string,
	parameters []string,
	envvar map[string]string,
	importModules []model.StorageSpec,
) SpecOpt {
	return func(s *model.Spec) error {
		// See wazero.ModuleConfig.WithEnv
		for key, value := range envvar {
			for _, str := range []string{key, value} {
				if str == "" || strings.ContainsRune(str, null) {
					return fmt.Errorf("invalid environment variable %s=%s", key, value)
				}
			}
		}
		s.Engine = model.EngineWasm
		s.Wasm = model.JobSpecWasm{
			EntryModule:          entryModule,
			EntryPoint:           entrypoint,
			Parameters:           parameters,
			EnvironmentVariables: envvar,
			ImportModules:        importModules,
		}
		return nil
	}
}
func MakeWasmSpec(
	entryModule model.StorageSpec, entrypoint string, parameters []string, envvar map[string]string, importModules []model.StorageSpec,
	opts ...SpecOpt,
) (model.Spec, error) {
	spec, err := MakeSpec(append(opts, WithWasmEngine(entryModule, entrypoint, parameters, envvar, importModules))...)
	if err != nil {
		return model.Spec{}, err
	}
	return spec, nil
}

// TODO(forrest): find a home
const DefaultTimeout = 30 * time.Minute

func MakeSpec(opts ...SpecOpt) (model.Spec, error) {
	spec := &model.Spec{
		Engine:    model.EngineNoop,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		PublisherSpec: model.PublisherSpec{
			Type: model.PublisherNoop,
		},
		Resources: model.ResourceUsageConfig{},
		Network: model.NetworkConfig{
			Type: model.NetworkNone,
		},
		Timeout: float64(DefaultTimeout),
		Deal: model.Deal{
			Concurrency: 1,
		},
	}

	for _, opt := range opts {
		if err := opt(spec); err != nil {
			return model.Spec{}, err
		}
	}

	return *spec, nil
}
