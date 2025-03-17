package testdata

import (
	_ "embed"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

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

//go:embed jobs/nameless.yaml
var namelessJobYAML []byte

//go:embed jobs/noop.yaml
var noopJobYAML []byte

//go:embed jobs/wasm.yaml
var wasmJobYAML []byte

//go:embed jobs/wasm_legacy.yaml
var wasmLegacyJobYAML []byte

var (
	DockerJobYAML    *Fixture
	DockerOutputYAML *Fixture
	DockerOutputJSON *Fixture
	DockerS3YAML     *Fixture
	EmptyJobYAML     *Fixture
	NamelessJobYAML  *Fixture
	NoopJobYAML      *Fixture
	WasmJobYAML      *Fixture
	WASMJobLegacy    *Fixture
)

func init() {
	DockerJobYAML = NewJobFixture("docker job", dockerJobYAML, false)
	DockerOutputYAML = NewJobFixture("docker with output yaml", dockerOutputYAML, false)
	DockerOutputJSON = NewJobFixture("docker with output json", dockerOutputJSON, false)
	DockerS3YAML = NewJobFixture("docker with s3", dockerS3YAML, false)
	EmptyJobYAML = NewJobFixture("empty job", emptyJobYAML, true)
	NamelessJobYAML = NewJobFixture("nameless job", namelessJobYAML, false)
	NoopJobYAML = NewJobFixture("noop", noopJobYAML, false)
	WasmJobYAML = NewJobFixture("wasm", wasmJobYAML, false)
	WASMJobLegacy = NewJobFixture("wasm legacy", wasmLegacyJobYAML, false)
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
		NamelessJobYAML,
		NoopJobYAML,
		WasmJobYAML,
		WASMJobLegacy,
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
