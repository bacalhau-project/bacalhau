//go:build unit || !integration

package docker

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	dm "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ExecuteCommand is a helper function to execute a command
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
	expectedDefaultNetworkConfig = &models.NetworkConfig{
		Type:    models.NetworkNone,
		Domains: []string(nil),
	}
	expectedDefaultTimeoutConfig = &models.TimeoutConfig{
		ExecutionTimeout: 0,
	}
)

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name          string
		flags         []string
		assertJob     func(t *testing.T, j *models.Job)
		expectedError bool
	}{
		{
			name:  "defaults with image",
			flags: []string{"image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)
				ds, err := dm.DecodeSpec(j.Task().Engine)
				require.NoError(t, err)

				assert.Equal(t, "image:tag", ds.Image)

				assert.Empty(t, ds.EnvironmentVariables)
				assert.Empty(t, ds.WorkingDirectory)
				assert.Empty(t, ds.Entrypoint)
				assert.Empty(t, ds.Parameters)
			},
			expectedError: false,
		},
		{
			name:  "with working dir",
			flags: []string{"image:tag", "--workdir=/dir"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)

				ds, err := dm.DecodeSpec(j.Task().Engine)
				require.NoError(t, err)
				assert.Equal(t, "image:tag", ds.Image)
				assert.Equal(t, "/dir", ds.WorkingDirectory)

				assert.Empty(t, ds.EnvironmentVariables)
				assert.Empty(t, ds.Entrypoint)
				assert.Empty(t, ds.Parameters)
			},
			expectedError: false,
		},
		{
			name:          "with invalid working dir",
			flags:         []string{"image:tag", "--workdir=dir"},
			expectedError: true,
		},
		{
			name:  "with env var",
			flags: []string{"--env=FOO=bar", "--env", "BAZ=buz", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)
				ds, err := dm.DecodeSpec(j.Task().Engine)
				require.NoError(t, err)

				assert.Equal(t, "image:tag", ds.Image)
				require.Len(t, ds.EnvironmentVariables, 2)
				assert.Contains(t, ds.EnvironmentVariables, "FOO=bar", "BAZ=buz")

				assert.Empty(t, ds.WorkingDirectory)
				assert.Empty(t, ds.Entrypoint)
				assert.Empty(t, ds.Parameters)
			},
			expectedError: false,
		},
		{
			name:  "with entry point",
			flags: []string{"--entrypoint=bin", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				defaultTaskAssertions(t, task)
				ds, err := dm.DecodeSpec(j.Task().Engine)
				require.NoError(t, err)

				assert.Equal(t, "image:tag", ds.Image)
				assert.Equal(t, []string{"bin"}, ds.Entrypoint)

				assert.Empty(t, ds.EnvironmentVariables)
				assert.Empty(t, ds.WorkingDirectory)
				assert.Empty(t, ds.Parameters)
			},
			expectedError: false,
		},
		{
			name:  "with name",
			flags: []string{"--name=testing", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, "testing", j.Name)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with namespace",
			flags: []string{"--namespace=testing", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, "testing", j.Namespace)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with priority",
			flags: []string{"--priority=1", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.EqualValues(t, 1, j.Priority)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with count",
			flags: []string{"--count=100", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.EqualValues(t, 100, j.Count)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with type batch",
			flags: []string{"--type=batch", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, models.JobTypeBatch, j.Type)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with type ops",
			flags: []string{"--type=ops", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, models.JobTypeOps, j.Type)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with type service",
			flags: []string{"--type=service", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, models.JobTypeService, j.Type)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with type daemon",
			flags: []string{"--type=daemon", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, models.JobTypeDaemon, j.Type)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:          "with invalid type",
			flags:         []string{"--type=invalid", "image:tag"},
			expectedError: true,
		},
		{
			name:  "with labels",
			flags: []string{"--labels", "foo=bar", "-l", "baz=buz", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Len(t, j.Labels, 2)
				assert.Equal(t, j.Labels["foo"], "bar")
				assert.Equal(t, j.Labels["baz"], "buz")
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:          "with invalid label key empty",
			flags:         []string{"--labels", "=bar", "-l", "baz=buz", "image:tag"},
			expectedError: true,
		},
		{
			name:          "with invalid label key not alphanumeric",
			flags:         []string{"--labels", "🐟=bar", "-l", "baz=buz", "image:tag"},
			expectedError: true,
		},
		{
			name:          "with invalid label value not alphanumeric",
			flags:         []string{"--labels", "foo=🐟", "-l", "baz=buz", "image:tag"},
			expectedError: true,
		},
		{
			name: "with constraints",
			flags: []string{"--constraints", "scale!=linear,env=prod,region notin (Utumno, Mordor, Isengard),size>42",
				"image:tag"},
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
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := NewCmd()
			output, err := ExecuteDryRun(cmd, test.flags...)
			if test.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, output)

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
	assert.Empty(t, task.Publisher.Type)
	assert.Empty(t, task.Publisher.Params)
	assert.Empty(t, task.Env)
	assert.Empty(t, task.Meta)
	assert.Empty(t, task.InputSources)
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
