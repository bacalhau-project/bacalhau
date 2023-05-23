package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/ipld/go-ipld-prime/datamodel"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine"
)

type TaskType string

const (
	TaskTypeDocker TaskType = "docker/run"
	TaskTypeWasm   TaskType = "wasm32-wasi/run"
	TaskTypeNoop   TaskType = "noop"
)

type Task struct {
	With   string
	Do     TaskType
	Inputs datamodel.Node
	Meta   IPLDMap[string, datamodel.Node]
}

type Resource struct {
	IPFS *IPFSResource
	HTTP *HTTPResource
}

type IPFSResource string
type HTTPResource string

type BacalhauConfig struct {
	Publisher   Publisher
	Verifier    Verifier
	Timeout     time.Duration
	Resources   ResourceSpec
	Annotations []string
	Dnt         bool
}

type ResourceSpec struct {
	Cpu    Millicores //nolint:stylecheck // name required by IPLD
	Disk   datasize.ByteSize
	Memory datasize.ByteSize
	Gpu    int
}

type JobType interface {
	EngineSpec(with string) (engine.Spec, error)
	InputStorageSpecs(with string) ([]StorageSpec, error)
	OutputStorageSpecs(with string) ([]StorageSpec, error)
}

type NoopTask struct{}

func (n NoopTask) EngineSpec(with string) (engine.Spec, error) {
	panic("TODO")
}

func (n NoopTask) InputStorageSpecs(with string) ([]StorageSpec, error) {
	return nil, nil
}

func (n NoopTask) OutputStorageSpecs(with string) ([]StorageSpec, error) {
	return nil, nil
}

var _ JobType = (*NoopTask)(nil)

func (task *Task) ToSpec() (*Spec, error) {
	var inputs JobType
	var err error
	switch task.Do {
	case TaskTypeDocker:
		inputs, err = Reinterpret[DockerInputs](task.Inputs, BacalhauTaskSchema)
	case TaskTypeWasm:
		inputs, err = Reinterpret[WasmInputs](task.Inputs, BacalhauTaskSchema)
	case TaskTypeNoop:
		inputs = NoopTask{}
	default:
		return nil, fmt.Errorf("TODO: task type %q", task.Do)
	}
	if err != nil {
		return nil, err
	}

	engineSpec, err := inputs.EngineSpec(task.With)
	if err != nil {
		return nil, err
	}
	inputSpec, err := inputs.InputStorageSpecs(task.With)
	if err != nil {
		return nil, err
	}
	outputSpec, err := inputs.OutputStorageSpecs(task.With)
	if err != nil {
		return nil, err
	}
	spec := new(Spec)
	spec.EngineSpec = engineSpec
	spec.Inputs = inputSpec
	spec.Outputs = outputSpec

	for key, node := range task.Meta.Values {
		switch key {
		case "bacalhau/config":
			config, err := Reinterpret[BacalhauConfig](node, BacalhauTaskSchema)
			if err != nil {
				return nil, err
			}

			spec.Verifier = config.Verifier
			spec.Publisher = config.Publisher
			spec.Annotations = config.Annotations
			spec.Timeout = config.Timeout.Seconds()
			spec.Resources = ResourceUsageConfig{
				CPU:    config.Resources.Cpu.String(),
				Memory: config.Resources.Memory.String(),
				Disk:   config.Resources.Disk.String(),
				GPU:    fmt.Sprint(config.Resources.Gpu),
			}
			spec.DoNotTrack = config.Dnt
		default:
			return nil, fmt.Errorf("TODO: config type %q", key)
		}
	}

	return spec, nil
}

func parseStorageSource(path string, resource *Resource) StorageSpec {
	storageSpec := StorageSpec{Path: path}
	if resource.IPFS != nil {
		storageSpec.StorageSource = StorageSourceIPFS
		storageSpec.CID = strings.TrimLeft(string(*resource.IPFS), ":/")
	} else if resource.HTTP != nil && strings.HasSuffix(path, ".git") {
		storageSpec.StorageSource = StorageSourceRepoClone
		storageSpec.Repo = "http" + string(*resource.HTTP)
	} else if resource.HTTP != nil {
		storageSpec.StorageSource = StorageSourceURLDownload
		storageSpec.URL = "http" + string(*resource.HTTP)
	}
	return storageSpec
}

func parseInputs(mounts IPLDMap[string, Resource]) ([]StorageSpec, error) {
	inputs := []StorageSpec{}
	for path, resource := range mounts.Values {
		resource := resource
		inputs = append(inputs, parseStorageSource(path, &resource))
	}
	return inputs, nil
}

func parseResource(uri string) (*Resource, error) {
	return UnmarshalIPLD[Resource]([]byte(`"`+uri+`"`), json.Decode, BacalhauTaskSchema)
}
