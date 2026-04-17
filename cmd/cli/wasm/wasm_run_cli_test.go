//go:build unit || !integration

/* spell-checker: disable */

package wasm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/cli/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ExecuteDryRun is a helper function to execute a command
func ExecuteDryRun(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	finalArgs := []string{"run", "--dry-run"}
	finalArgs = append(finalArgs, args...)
	root.SetArgs(finalArgs)

	_, err = root.ExecuteC()
	return buf.String(), err
}

var (
	expectedDefaultResourceConfig = &models.ResourcesConfig{
		CPU:    "",
		Memory: "",
		Disk:   "",
		GPU:    "",
	}
	expectedDefaultNetworkConfig = (*models.NetworkConfig)(nil)
	expectedDefaultTimeoutConfig = &models.TimeoutConfig{
		TotalTimeout: 0,
		QueueTimeout: 0,
	}
)

func TestJobFlagParsing(t *testing.T) {
	// disable the update checker in testing
	t.Setenv(config.KeyAsEnvVar(types.UpdateConfigIntervalKey), "0")

	repoPath := t.TempDir()
	viper.Set("repo", repoPath)
	tests := []struct {
		name          string
		flags         []string
		assertJob     func(t *testing.T, j *models.Job)
		expectedError bool
	}{
		{
			name:  "local module with default target",
			flags: []string{"../../../testdata/wasm/noop/main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "main.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Empty(t, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				require.Len(t, task.InputSources, 1)
				assert.Equal(t, "main.wasm", task.InputSources[0].Target)
				assert.Equal(t, models.StorageSourceInline, task.InputSources[0].Source.Type)
			},
			expectedError: false,
		},
		{
			name:  "local module with custom target",
			flags: []string{"../../../testdata/wasm/noop/main.wasm:/app/custom.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "/app/custom.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Empty(t, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				require.Len(t, task.InputSources, 1)
				assert.Equal(t, "/app/custom.wasm", task.InputSources[0].Target)
				assert.Equal(t, models.StorageSourceInline, task.InputSources[0].Source.Type)
			},
			expectedError: false,
		},
		{
			name:  "remote module with default target",
			flags: []string{"https://example.com/main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "main.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Empty(t, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				require.Len(t, task.InputSources, 1)
				assert.Equal(t, "main.wasm", task.InputSources[0].Target)
				assert.Equal(t, models.StorageSourceURL, task.InputSources[0].Source.Type)
				assert.Equal(t, "https://example.com/main.wasm", task.InputSources[0].Source.Params["URL"])
			},
			expectedError: false,
		},
		{
			name:  "remote module with custom target",
			flags: []string{"https://example.com/main.wasm:/app/custom.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "/app/custom.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Empty(t, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				require.Len(t, task.InputSources, 1)
				assert.Equal(t, "/app/custom.wasm", task.InputSources[0].Target)
				assert.Equal(t, models.StorageSourceURL, task.InputSources[0].Source.Type)
				assert.Equal(t, "https://example.com/main.wasm", task.InputSources[0].Source.Params["URL"])
			},
			expectedError: false,
		},
		{
			name:  "non-existent local module (pass as is)",
			flags: []string{"./nonexistent.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "./nonexistent.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Empty(t, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				require.Len(t, task.InputSources, 0)
			},
			expectedError: false,
		},
		{
			name:          "with local import module without target",
			flags:         []string{"--import-modules", "../../../testdata/wasm/env/main.wasm", "../../../testdata/wasm/noop/main.wasm"},
			expectedError: true,
		},
		{
			name:          "with remote import module without target",
			flags:         []string{"--import-modules", "https://example.com/lib.wasm", "https://example.com/main.wasm"},
			expectedError: true,
		},
		{
			name:  "with local import module with target",
			flags: []string{"--import-modules", "../../../testdata/wasm/env/main.wasm:/app/lib.wasm", "../../../testdata/wasm/noop/main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "main.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Equal(t, []interface{}{"/app/lib.wasm"}, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				// Create a map of target to input source for order-independent comparison
				inputSources := make(map[string]models.InputSource)
				for _, source := range task.InputSources {
					inputSources[source.Target] = *source
				}

				require.Len(t, inputSources, 2)
				assert.Equal(t, "/app/lib.wasm", inputSources["/app/lib.wasm"].Target)
				assert.Equal(t, models.StorageSourceInline, inputSources["/app/lib.wasm"].Source.Type)
				assert.Equal(t, "main.wasm", inputSources["main.wasm"].Target)
				assert.Equal(t, models.StorageSourceInline, inputSources["main.wasm"].Source.Type)
			},
			expectedError: false,
		},
		{
			name:  "with remote import module with target",
			flags: []string{"--import-modules", "https://example.com/lib.wasm:/app/lib.wasm", "https://example.com/main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "main.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Equal(t, []interface{}{"/app/lib.wasm"}, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				// Create a map of target to input source for order-independent comparison
				inputSources := make(map[string]models.InputSource)
				for _, source := range task.InputSources {
					inputSources[source.Target] = *source
				}

				require.Len(t, inputSources, 2)
				assert.Equal(t, "/app/lib.wasm", inputSources["/app/lib.wasm"].Target)
				assert.Equal(t, models.StorageSourceURL, inputSources["/app/lib.wasm"].Source.Type)
				assert.Equal(t, "https://example.com/lib.wasm", inputSources["/app/lib.wasm"].Source.Params["URL"])
				assert.Equal(t, "main.wasm", inputSources["main.wasm"].Target)
				assert.Equal(t, models.StorageSourceURL, inputSources["main.wasm"].Source.Type)
				assert.Equal(t, "https://example.com/main.wasm", inputSources["main.wasm"].Source.Params["URL"])
			},
			expectedError: false,
		},
		{
			name: "with multiple import modules",
			flags: []string{
				"--import-modules", "https://example.com/lib1.wasm:/app/lib1.wasm",
				"--import-modules", "https://example.com/lib2.wasm:/app/lib2.wasm",
				"https://example.com/main.wasm",
			},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				assert.Equal(t, models.EngineWasm, task.Engine.Type)
				assert.Equal(t, "main.wasm", task.Engine.Params["EntryModule"])
				assert.Equal(t, "_start", task.Engine.Params["Entrypoint"])
				assert.Equal(t, []interface{}{"/app/lib1.wasm", "/app/lib2.wasm"}, task.Engine.Params["ImportModules"])
				assert.Empty(t, task.Engine.Params["Parameters"])

				// Create a map of target to input source for order-independent comparison
				inputSources := make(map[string]models.InputSource)
				for _, source := range task.InputSources {
					inputSources[source.Target] = *source
				}

				require.Len(t, inputSources, 3)
				assert.Equal(t, "/app/lib1.wasm", inputSources["/app/lib1.wasm"].Target)
				assert.Equal(t, models.StorageSourceURL, inputSources["/app/lib1.wasm"].Source.Type)
				assert.Equal(t, "https://example.com/lib1.wasm", inputSources["/app/lib1.wasm"].Source.Params["URL"])
				assert.Equal(t, "/app/lib2.wasm", inputSources["/app/lib2.wasm"].Target)
				assert.Equal(t, models.StorageSourceURL, inputSources["/app/lib2.wasm"].Source.Type)
				assert.Equal(t, "https://example.com/lib2.wasm", inputSources["/app/lib2.wasm"].Source.Params["URL"])
				assert.Equal(t, "main.wasm", inputSources["main.wasm"].Target)
				assert.Equal(t, models.StorageSourceURL, inputSources["main.wasm"].Source.Type)
				assert.Equal(t, "https://example.com/main.wasm", inputSources["main.wasm"].Source.Params["URL"])
			},
			expectedError: false,
		},
		{
			name:  "with name",
			flags: []string{"--name=test-job", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, "test-job", j.Name)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with namespace",
			flags: []string{"--namespace=test-ns", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, "test-ns", j.Namespace)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with priority",
			flags: []string{"--priority=1", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.EqualValues(t, 1, j.Priority)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with count",
			flags: []string{"--count=100", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.EqualValues(t, 100, j.Count)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with concurrency",
			flags: []string{"--concurrency=100", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.EqualValues(t, 100, j.Count)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with network none",
			flags: []string{"--network=none", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkNone, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:  "with network http",
			flags: []string{"--network=http", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkHTTP, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:  "with network host",
			flags: []string{"--network=host", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkHost, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:          "with network invalid",
			flags:         []string{"--network=invalid", "./main.wasm"},
			expectedError: true,
		},
		{
			name:  "with timeout",
			flags: []string{"--timeout=300", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.EqualValues(t, 300, task.Timeouts.TotalTimeout)
			},
			expectedError: false,
		},
		{
			name:  "with queue timeout",
			flags: []string{"--queue-timeout=300", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.EqualValues(t, 300, task.Timeouts.QueueTimeout)
			},
			expectedError: false,
		},
		{
			name:  "with labels",
			flags: []string{"--labels", "foo=bar", "-l", "baz=buz", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Len(t, j.Labels, 2)
				assert.Equal(t, j.Labels["foo"], "bar")
				assert.Equal(t, j.Labels["baz"], "buz")
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with constraints",
			flags: []string{"--constraints", "scale!=linear,env=prod,region notin (Utumno, Mordor, Isengard),size>42", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				require.Len(t, j.Constraints, 4)
				// constraints are always alphanumerically sorted.
				expected := []struct {
					key      string
					operator string
					values   []string
				}{
					{
						"env",
						"=",
						[]string{"prod"},
					},
					{
						"region",
						"notin",
						[]string{"Isengard", "Mordor", "Utumno"},
					},
					{
						"scale",
						"!=",
						[]string{"linear"},
					},
					{
						"size",
						"gt",
						[]string{"42"},
					},
				}
				for i, expect := range expected {
					assert.Equal(t, expect.key, j.Constraints[i].Key)
					assert.EqualValues(t, expect.operator, j.Constraints[i].Operator)
					assert.Len(t, j.Constraints[i].Values, len(expect.values))
					assert.EqualValues(t, expect.values, j.Constraints[i].Values)
				}
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with resources",
			flags: []string{"--cpu=1", "--memory=2", "--disk=3", "--gpu=4", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, "1", task.ResourcesConfig.CPU)
				assert.Equal(t, "2", task.ResourcesConfig.Memory)
				assert.Equal(t, "3", task.ResourcesConfig.Disk)
				assert.Equal(t, "4", task.ResourcesConfig.GPU)
			},
			expectedError: false,
		},
		{
			name:  "with task-name",
			flags: []string{"--task-name=TASKNAME", "./main.wasm"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, "TASKNAME", task.Name)
			},
			expectedError: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := wasm.NewCmd()
			output, err := ExecuteDryRun(cmd, test.flags...)
			if test.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, output)

				// NB(forrest): because https://github.com/spf13/cobra/issues/1708
				// deprecation warnings are printed to stdout instead of stderr
				if strings.Contains(output, "has been deprecated") {
					lines := strings.Split(output, "\n")
					require.NotEmpty(t, lines)
					output = strings.Join(lines[1:], "\n")
				}
				actual := new(models.Job)
				err = yaml.Unmarshal([]byte(output), actual)
				require.NoError(t, err)

				test.assertJob(t, actual)
			}
		})
	}
}

func defaultTaskAssertions(t *testing.T, task *models.Task) {
	assert.Equal(t, task.Name, "main")
	assert.Nil(t, task.Publisher)
	assert.Empty(t, task.Env)
	assert.Empty(t, task.Meta)
	assert.Empty(t, task.ResultPaths)
	assert.Equal(t, expectedDefaultResourceConfig, task.ResourcesConfig)
	assert.Equal(t, expectedDefaultNetworkConfig, task.Network)
	assert.Equal(t, expectedDefaultTimeoutConfig, task.Timeouts)
}

func defaultJobAssertions(t *testing.T, j *models.Job) {
	assert.Empty(t, j.ID)
	assert.Empty(t, j.Name)
	assert.Equal(t, models.DefaultNamespace, j.Namespace)
	assert.Equal(t, models.JobTypeBatch, j.Type)
	assert.Zero(t, j.Priority)
	assert.Equal(t, 1, j.Count)
	assert.Empty(t, j.Constraints)
	assert.Empty(t, j.Meta)
	assert.Empty(t, j.Labels)
	assert.Equal(t, models.JobStateTypeUndefined, j.State.StateType)
	assert.Empty(t, j.State.Message)
	assert.Zero(t, j.Version)
	assert.Zero(t, j.Revision)
	assert.Zero(t, j.CreateTime)
	assert.Zero(t, j.ModifyTime)
	assert.Len(t, j.Tasks, 1)
}
