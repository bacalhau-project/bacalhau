package testdata

import (
	_ "embed"
	"fmt"
	"os"
	"testing"

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

var (
	JsonJobNoop   *Fixture
	JsonJobCancel *Fixture

	YamlJobS3          *Fixture
	YamlJobNoop        *Fixture
	YamlJobNoopInvalid *Fixture
	YamlJobNoopUrl     *Fixture

	IPVMTaskDocker     *Fixture
	IPVMTaskWasm       *Fixture
	IPVMTaskWithConfig *Fixture
)

func init() {
	JsonJobNoop = NewSpecFixture(jobNoopJSON)
	YamlJobNoop = NewSpecFixture(jobNoopYAML)
	YamlJobNoopInvalid = NewSpecFixture(jobNoopYAMLInvalid)

	JsonJobCancel = NewSpecFixture(jobCancelJSON)

	YamlJobS3 = NewSpecFixture(jobS3YAML)

	YamlJobNoopUrl = NewSpecFixture(jobNoopURLYAML)

	IPVMTaskDocker = NewIPVMFixture(dockerTaskJSON)
	IPVMTaskWasm = NewIPVMFixture(wasmTaskJSON)
	IPVMTaskWithConfig = NewIPVMFixture(taskWithConfigJSON)

}

type Fixture struct {
	Job  model.Job
	Data []byte
}

func (f *Fixture) RequiresDocker() bool {
	return f.Job.Spec.Engine == model.EngineDocker
}

func (f *Fixture) RequiresS3() bool {
	for _, i := range f.Job.Spec.AllStorageSpecs() {
		if i.StorageSource == model.StorageSourceS3 {
			return true
		}
	}
	return false
}

// validate validates the fixture.
func (f *Fixture) validate() {
	// validate the job spec was deserialized correctly and not empty
	// checking for valid engine seems like a good enough check
	if !model.IsValidEngine(f.Job.Spec.Engine) {
		panic(fmt.Errorf("spec is empty/invalid: %s", string(f.Data)))
	}
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

func NewSpecFixture(data []byte) *Fixture {
	var out model.Job
	if err := marshaller.YAMLUnmarshalWithMax(data, &out); err != nil {
		panic(err)
	}

	f := &Fixture{
		Job:  out,
		Data: data,
	}
	f.validate()
	return f
}

func NewIPVMFixture(data []byte) *Fixture {
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

	f := &Fixture{
		Job:  *job,
		Data: data,
	}
	f.validate()
	return f
}
