package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type SpecOpt func(s *v1beta2.Spec) error

func WithVerifier(v v1beta2.Verifier) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Verifier = v
		return nil
	}
}

func WithPublisher(p v1beta2.PublisherSpec) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Publisher = p.Type
		s.PublisherSpec = p
		return nil
	}
}

func WithNetwork(network v1beta2.Network, domains []string) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Network.Type = network
		s.Network.Domains = domains
		return nil
	}
}

func WithResources(cpu, memory, disk, gpu string) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Resources.CPU = cpu
		s.Resources.Memory = memory
		s.Resources.Disk = disk
		s.Resources.GPU = gpu
		return nil
	}
}

func WithTimeout(t float64) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Timeout = t
		return nil
	}
}

func WithDeal(targeting v1beta2.TargetingMode, concurrency, confidence, minbids int) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Deal.TargetingMode = targeting
		s.Deal.Concurrency = concurrency
		s.Deal.Confidence = confidence
		s.Deal.MinBids = minbids
		return nil
	}
}

func WithAnnotations(annotations ...string) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Annotations = annotations
		return nil
	}
}

func WithInputs(inputs ...v1beta2.StorageSpec) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Inputs = inputs
		return nil
	}
}

func WithOutputs(outputs ...v1beta2.StorageSpec) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.Outputs = outputs
		return nil
	}
}

func WithNodeSelector(selector []v1beta2.LabelSelectorRequirement) SpecOpt {
	return func(s *v1beta2.Spec) error {
		s.NodeSelectors = selector
		return nil
	}
}

func WithDockerEngine(image, workdir string, entrypoint, envvar, parameters []string) SpecOpt {
	return func(s *v1beta2.Spec) error {
		if err := system.ValidateWorkingDir(workdir); err != nil {
			return fmt.Errorf("validating docker working directory: %w", err)
		}
		s.Engine = v1beta2.EngineDocker
		s.Docker = v1beta2.JobSpecDocker{
			Image:                image,
			Entrypoint:           entrypoint,
			Parameters:           parameters,
			EnvironmentVariables: envvar,
			WorkingDirectory:     workdir,
		}
		return nil
	}
}

func MakeDockerSpec(
	image, workingdir string, entrypoint, envvar, parameters []string,
	opts ...SpecOpt,
) (v1beta2.Spec, error) {
	spec, err := MakeSpec(append(opts, WithDockerEngine(image, workingdir, entrypoint, envvar, parameters))...)
	if err != nil {
		return v1beta2.Spec{}, err
	}
	return spec, nil
}

const null rune = 0

func WithWasmEngine(
	entryModule v1beta2.StorageSpec,
	entrypoint string,
	parameters []string,
	envvar map[string]string,
	importModules []v1beta2.StorageSpec,
) SpecOpt {
	return func(s *v1beta2.Spec) error {
		// See wazero.ModuleConfig.WithEnv
		for key, value := range envvar {
			for _, str := range []string{key, value} {
				if str == "" || strings.ContainsRune(str, null) {
					return fmt.Errorf("invalid environment variable %s=%s", key, value)
				}
			}
		}
		s.Engine = v1beta2.EngineWasm
		s.Wasm = v1beta2.JobSpecWasm{
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
	entryModule v1beta2.StorageSpec, entrypoint string, parameters []string, envvar map[string]string, importModules []v1beta2.StorageSpec,
	opts ...SpecOpt,
) (v1beta2.Spec, error) {
	spec, err := MakeSpec(append(opts, WithWasmEngine(entryModule, entrypoint, parameters, envvar, importModules))...)
	if err != nil {
		return v1beta2.Spec{}, err
	}
	return spec, nil
}

// TODO(forrest): find a home
const DefaultTimeout = 30 * time.Minute

func MakeSpec(opts ...SpecOpt) (v1beta2.Spec, error) {
	spec := &v1beta2.Spec{
		Engine:    v1beta2.EngineNoop,
		Verifier:  v1beta2.VerifierNoop,
		Publisher: v1beta2.PublisherNoop,
		PublisherSpec: v1beta2.PublisherSpec{
			Type: v1beta2.PublisherNoop,
		},
		Resources: v1beta2.ResourceUsageConfig{},
		Network: v1beta2.NetworkConfig{
			Type: v1beta2.NetworkNone,
		},
		Timeout: float64(DefaultTimeout),
		Deal: v1beta2.Deal{
			Concurrency: 1,
		},
	}

	for _, opt := range opts {
		if err := opt(spec); err != nil {
			return v1beta2.Spec{}, err
		}
	}

	return *spec, nil
}
