package testdata

import (
	_ "embed"
	"os"
	"testing"

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

	TaskDockerJson     *Fixture
	TaskWasmJson       *Fixture
	TaskWithConfigJson *Fixture
)

func init() {
	JsonJobNoop = NewSpecFixture(jobNoopJSON)
	YamlJobNoop = NewSpecFixture(jobNoopYAML)
	YamlJobNoopInvalid = NewSpecFixture(jobNoopYAMLInvalid)

	JsonJobCancel = NewSpecFixture(jobCancelJSON)

	YamlJobS3 = NewSpecFixture(jobS3YAML)

	YamlJobNoopUrl = NewSpecFixture(jobNoopURLYAML)

	TaskDockerJson = NewSpecFixture(dockerTaskJSON)
	TaskWasmJson = NewSpecFixture(wasmTaskJSON)
	TaskWithConfigJson = NewSpecFixture(taskWithConfigJSON)

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
	if err := model.YAMLUnmarshalWithMax(data, &out); err != nil {
		panic(err)
	}

	return &Fixture{
		Job:  out,
		Data: data,
	}
}
