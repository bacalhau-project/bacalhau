package testdata

import (
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

//go:embed job-noop.json
var jobNoopJSON []byte

//go:embed job-noop.yaml
var jobNoopYAML []byte

//go:embed job-noop-invalid.yml
var jobNoopYAMLInvalid []byte

//go:embed job_cancel.json
var jobCancelJSON []byte

//go:embed job-s3.yaml
var jobS3YAML []byte

//go:embed job-noop-url.yaml
var jobNoopURLYAML []byte

//go:embed docker_task.json
var dockerTaskJSON []byte

//go:embed task_with_config.json
var taskWithConfigJSON []byte

//go:embed wasm_task.json
var wasmTaskJSON []byte

//go:embed job-docker-engine-spec.json
var jobJsonDockerEngineSpec []byte

//go:embed job-docker-engine-spec.yaml
var jobYamlDockerEngineSpec []byte

//go:embed job-wasm-engine-spec.json
var jobJsonWasmEngineSpec []byte

//go:embed jobs/docker.yaml
var dockerJobYAML []byte

//go:embed jobs/docker-output.yaml
var dockerOutputYAML []byte

//go:embed jobs/docker-output.json
var dockerOutputJSON []byte

//go:embed jobs/docker-s3.yaml
var dockerS3YAML []byte

//go:embed jobs/empty.yaml
var emptyJobYAML []byte

//go:embed jobs/noop.yaml
var noopJobYAML []byte

//go:embed jobs/wasm.yaml
var wasmJobYAML []byte

var (
	JsonJobNoop             *FixtureLegacy
	JsonJobCancel           *FixtureLegacy
	JsonJobDockerEngineSpec *FixtureLegacy
	JsonJobWasmEngineSpec   *FixtureLegacy

	YamlJobS3               *FixtureLegacy
	YamlJobNoop             *FixtureLegacy
	YamlJobNoopInvalid      *FixtureLegacy
	YamlJobNoopUrl          *FixtureLegacy
	YamlJobDockerEngineSpec *FixtureLegacy

	IPVMTaskDocker     *FixtureLegacy
	IPVMTaskWasm       *FixtureLegacy
	IPVMTaskWithConfig *FixtureLegacy

	DockerJobYAML    *Fixture
	DockerOutputYAML *Fixture
	DockerOutputJSON *Fixture
	DockerS3YAML     *Fixture
	EmptyJobYAML     *Fixture
	NoopJobYAML      *Fixture
	WasmJobYAML      *Fixture
)

func init() {
	JsonJobNoop = NewLegacySpecFixture(jobNoopJSON)
	YamlJobNoop = NewLegacySpecFixture(jobNoopYAML)
	YamlJobNoopInvalid = NewLegacySpecFixture(jobNoopYAMLInvalid)

	JsonJobCancel = NewLegacySpecFixture(jobCancelJSON)

	YamlJobS3 = NewLegacySpecFixture(jobS3YAML)

	YamlJobNoopUrl = NewLegacySpecFixture(jobNoopURLYAML)

	IPVMTaskDocker = NewLegacyIPVMFixture(dockerTaskJSON)
	IPVMTaskWasm = NewLegacyIPVMFixture(wasmTaskJSON)
	IPVMTaskWithConfig = NewLegacyIPVMFixture(taskWithConfigJSON)

	JsonJobDockerEngineSpec = NewLegacySpecFixture(jobJsonDockerEngineSpec)
	YamlJobDockerEngineSpec = NewLegacySpecFixture(jobYamlDockerEngineSpec)

	JsonJobWasmEngineSpec = NewLegacySpecFixture(jobJsonWasmEngineSpec)

	DockerJobYAML = NewJobFixture("docker job", dockerJobYAML, false)
	DockerOutputYAML = NewJobFixture("docker with output yaml", dockerOutputYAML, false)
	DockerOutputJSON = NewJobFixture("docker with output json", dockerOutputJSON, false)
	DockerS3YAML = NewJobFixture("docker with s3", dockerS3YAML, false)
	EmptyJobYAML = NewJobFixture("empty job", emptyJobYAML, true)
	NoopJobYAML = NewJobFixture("noop", noopJobYAML, false)
	WasmJobYAML = NewJobFixture("wasm", wasmJobYAML, false)
}

type Fixture struct {
	Description string
	Job         *models.Job
	Data        []byte
	Invalid     bool
}

func AllFixtures() []*Fixture {
	return []*Fixture{
		DockerJobYAML,
		DockerOutputYAML,
		DockerOutputJSON,
		DockerS3YAML,
		EmptyJobYAML,
		NoopJobYAML,
		WasmJobYAML,
	}
}

func (f *Fixture) RequiresDocker() bool {
	if f.Job == nil {
		return false
	}
	return f.Job.Task().Engine.Type == models.EngineDocker
}

func (f *Fixture) RequiresS3() bool {
	if f.Job == nil {
		return false
	}
	for _, i := range f.Job.AllStorageTypes() {
		if i == models.StorageSourceS3 {
			return true
		}
	}
	if f.Job.Task().Publisher.IsType(models.PublisherS3) {
		return true
	}
	return false
}

func (f *Fixture) AsTempFile(t testing.TB, pattern string) string {
	tmpfile, err := os.CreateTemp("", pattern)
	require.NoError(t, err)

	_, err = tmpfile.Write(f.Data)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	return tmpfile.Name()
}

func NewJobFixture(description string, data []byte, invalid bool) *Fixture {
	out := new(models.Job)
	if err := marshaller.YAMLUnmarshalWithMax(data, &out); err != nil {
		panic(err)
	}

	return &Fixture{
		Description: description,
		Job:         out,
		Data:        data,
		Invalid:     invalid,
	}
}

type FixtureLegacy struct {
	Job  model.Job
	Data []byte
}

func (f *FixtureLegacy) RequiresDocker() bool {
	return f.Job.Spec.EngineSpec.Engine() == model.EngineDocker
}

func (f *FixtureLegacy) RequiresS3() bool {
	for _, i := range f.Job.Spec.AllStorageSpecs() {
		if i.StorageSource == model.StorageSourceS3 {
			return true
		}
	}
	return false
}

// validate validates the fixture.
func (f *FixtureLegacy) validate() {
	// validate the job spec was deserialized correctly and not empty
	// checking for valid engine seems like a good enough check
	if !model.IsValidEngine(f.Job.Spec.EngineSpec.Engine()) {
		panic(fmt.Errorf("spec is empty/invalid: %s", string(f.Data)))
	}
}

func (f *FixtureLegacy) AsTempFile(t testing.TB, pattern string) string {
	tmpfile, err := os.CreateTemp("", pattern)
	require.NoError(t, err)

	_, err = tmpfile.Write(f.Data)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	return tmpfile.Name()
}

func NewLegacySpecFixture(data []byte) *FixtureLegacy {
	var out model.Job
	if err := marshaller.YAMLUnmarshalWithMax(data, &out); err != nil {
		panic(err)
	}

	f := &FixtureLegacy{
		Job:  out,
		Data: data,
	}
	//f.validate()
	return f
}

func NewLegacyIPVMFixture(data []byte) *FixtureLegacy {
	task, err := model.UnmarshalIPLD[model.Task](data, json.Decode, model.UCANTaskSchema)
	if err != nil {
		panic(err)
	}
	spec, err := task.ToSpec()
	if err != nil {
		panic(err)
	}

	job, err := model.NewJobWithSaneProductionDefaults()
	if err != nil {
		panic(err)
	}
	job.Spec = *spec

	f := &FixtureLegacy{
		Job:  *job,
		Data: data,
	}
	//f.validate()
	return f
}
