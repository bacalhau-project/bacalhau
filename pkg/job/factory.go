package job

import (
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type SpecOpt func(s *model.Spec) error

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

func WithDockerEngine(image, workdir string, entrypoint, envvar []string) SpecOpt {
	return func(s *model.Spec) error {
		if err := system.ValidateWorkingDir(workdir); err != nil {
			return fmt.Errorf("validating docker working directory: %w", err)
		}
		s.EngineDeprecated = model.EngineDocker
		s.EngineSpec = model.NewDockerEngineSpec(image, entrypoint, envvar, workdir)
		return nil
	}
}

func MakeDockerSpec(
	image string, entrypoint []string, envvar []string, workingdir string,
	opts ...SpecOpt,
) (model.Spec, error) {
	spec, err := MakeSpec(append(opts, WithDockerEngine(image, workingdir, entrypoint, envvar))...)
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
		s.EngineDeprecated = model.EngineWasm
		s.EngineSpec = model.NewWasmEngineSpec(entryModule, entrypoint, parameters, envvar, importModules)
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
		EngineDeprecated: model.EngineNoop,
		EngineSpec: model.EngineSpec{
			Type: model.EngineNoop.String(),
		},
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
