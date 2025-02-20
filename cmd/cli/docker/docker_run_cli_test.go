//go:build unit || !integration

/* spell-checker: disable */

package docker

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	dm "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_s3 "github.com/bacalhau-project/bacalhau/pkg/s3"
	storage_ipfs "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	storage_s3 "github.com/bacalhau-project/bacalhau/pkg/storage/s3"
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
			name:  "with concurrency",
			flags: []string{"--concurrency=100", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.EqualValues(t, 100, j.Count)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with target all",
			flags: []string{"--target=all", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, models.JobTypeOps, j.Type)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:  "with target any",
			flags: []string{"--target=any", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Equal(t, models.JobTypeBatch, j.Type)
				defaultTaskAssertions(t, j.Task())
			},
			expectedError: false,
		},
		{
			name:          "with target invalid",
			flags:         []string{"--target=batch", "image:tag"},
			expectedError: true,
		},
		{
			name:  "with labels key value pairs",
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
			name:  "with labels keys only",
			flags: []string{"--labels", "foo", "-l", "baz", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				assert.Len(t, j.Labels, 2)
				fooEmpty, fooOk := j.Labels["foo"]
				assert.Empty(t, fooEmpty)
				assert.True(t, fooOk)
				bazEmpty, bazOk := j.Labels["baz"]
				assert.Empty(t, bazEmpty)
				assert.True(t, bazOk)
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
		{
			name: "with deprecated selector",
			flags: []string{"--selector", "scale!=linear,env=prod,region notin (Utumno, Mordor, Isengard),size>42",
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
		{
			name:  "with task-name",
			flags: []string{"--task-name=TASKNAME", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, "TASKNAME", task.Name)
			},
			expectedError: false,
		},
		{
			name:  "with resources",
			flags: []string{"--cpu=1", "--memory=2", "--disk=3", "--gpu=4", "image:tag"},
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
			name:  "with network none default",
			flags: []string{"image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkNone, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:  "with network none flag",
			flags: []string{"--network=none", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkNone, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:  "with network http",
			flags: []string{"--network=http", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkHTTP, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:  "with network full",
			flags: []string{"--network=full", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkHost, task.Network.Type)
			},
			expectedError: false,
		},
		{
			name:          "with network invalid",
			flags:         []string{"--network=invalid", "image:tag"},
			expectedError: true,
		},
		{
			name:  "with http network and domain",
			flags: []string{"--network=http", "--domain=example.com", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkHTTP, task.Network.Type)
				assert.Equal(t, []string{"example.com"}, task.Network.Domains)
			},
			expectedError: false,
		},
		{
			name:  "with http network and domains",
			flags: []string{"--network=http", "--domain=example.com", "--domain=example.io", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, models.NetworkHTTP, task.Network.Type)
				assert.Equal(t, []string{"example.com", "example.io"}, task.Network.Domains)
			},
			expectedError: false,
		},
		// TODO(forrest): if/when validation on the network config is adjusted expect this test to fail.
		{
			name:          "with none network and domains",
			flags:         []string{"--network=none", "--domain=example.com", "--domain=example.io", "image:tag"},
			expectedError: true,
		},
		{
			name:  "with timeout",
			flags: []string{"--timeout=300", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.EqualValues(t, 300, task.Timeouts.TotalTimeout)
			},
			expectedError: false,
		},
		{
			name:  "with queue timeout",
			flags: []string{"--queue-timeout=300", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.EqualValues(t, 300, task.Timeouts.QueueTimeout)
			},
			expectedError: false,
		},
		{
			name:  "with ipfs publisher",
			flags: []string{"--publisher=ipfs", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, task.Publisher.Type, models.PublisherIPFS)
				assert.Empty(t, task.Publisher.Params)
			},
			expectedError: false,
		},
		{
			name:  "with local publisher",
			flags: []string{"--publisher=local", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				assert.Equal(t, task.Publisher.Type, models.PublisherLocal)
				assert.Empty(t, task.Publisher.Params)
			},
			expectedError: false,
		},
		{
			name:  "with s3 publisher",
			flags: []string{"--publisher=s3://myBucket/myKey", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				s3publisher, err := publisher_s3.DecodePublisherSpec(task.Publisher)
				require.NoError(t, err)
				assert.Equal(t, publisher_s3.PublisherSpec{
					Bucket: "myBucket",
					Key:    "myKey",
				}, s3publisher)
			},
			expectedError: false,
		},
		{
			name:  "with s3 publisher with opts",
			flags: []string{"-p=s3://myBucket/myKey,opt=region=us-west-2,opt=endpoint=https://s3.custom.com", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				s3publisher, err := publisher_s3.DecodePublisherSpec(task.Publisher)
				require.NoError(t, err)
				assert.Equal(t, publisher_s3.PublisherSpec{
					Bucket:   "myBucket",
					Key:      "myKey",
					Region:   "us-west-2",
					Endpoint: "https://s3.custom.com",
				}, s3publisher)
			},
			expectedError: false,
		},
		{
			name:  "with s3 publisher with options",
			flags: []string{"-p=s3://myBucket/myKey,option=region=us-west-2,option=endpoint=https://s3.custom.com", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				s3publisher, err := publisher_s3.DecodePublisherSpec(task.Publisher)
				require.NoError(t, err)
				assert.Equal(t, publisher_s3.PublisherSpec{
					Bucket:   "myBucket",
					Key:      "myKey",
					Region:   "us-west-2",
					Endpoint: "https://s3.custom.com",
				}, s3publisher)
			},
			expectedError: false,
		},
		{
			name:          "with unknown with options",
			flags:         []string{"-p=abc://123", "image:tag"},
			expectedError: true,
		},
		{
			name:  "with ipfs input",
			flags: []string{"--input=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", inputSource.Alias)
				assert.Equal(t, "/inputs", inputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)
			},
			expectedError: false,
		},
		{
			name:  "with ipfs with path",
			flags: []string{"--input=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf:/mount/path", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", inputSource.Alias)
				assert.Equal(t, "/mount/path", inputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)
			},
			expectedError: false,
		},
		{
			name:  "with ipfs input and target",
			flags: []string{"--input=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf,target=/target", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", inputSource.Alias)
				assert.Equal(t, "/target", inputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)
			},
			expectedError: false,
		},
		{
			name:  "with ipfs input and dst",
			flags: []string{"--input=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf,dst=/target", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", inputSource.Alias)
				assert.Equal(t, "/target", inputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)
			},
			expectedError: false,
		},
		{
			name:  "with ipfs input and destination",
			flags: []string{"--input=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf,destination=/target", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", inputSource.Alias)
				assert.Equal(t, "/target", inputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)
			},
			expectedError: false,
		},
		{
			name:  "with ipfs input with explicit src and dst",
			flags: []string{"--input=src=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf,dst=/mount/path", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", inputSource.Alias)
				assert.Equal(t, "/mount/path", inputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)
			},
			expectedError: false,
		},
		{
			name:  "with s3 publisher",
			flags: []string{"--input=s3://myBucket/dir/file-001.txt", "image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 1)
				inputSource := task.InputSources[0]
				s3Input, err := storage_s3.DecodeSourceSpec(inputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "s3://myBucket/dir/file-001.txt", inputSource.Alias)
				assert.Equal(t, "/inputs", inputSource.Target)
				assert.Equal(t, storage_s3.SourceSpec{
					Bucket: "myBucket",
					Key:    "dir/file-001.txt",
				}, s3Input)
			},
			expectedError: false,
		},
		{
			name: "with s3 and IPFS publisher",
			flags: []string{
				"-i=ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf,destination=/target",
				"--input=s3://myBucket/dir/file-001.txt",
				"image:tag"},
			assertJob: func(t *testing.T, j *models.Job) {
				defaultJobAssertions(t, j)
				task := j.Task()
				require.Len(t, task.InputSources, 2)

				ipfsInputSource := task.InputSources[0]
				ipfsInput, err := storage_ipfs.DecodeSpec(ipfsInputSource.Source)
				require.NoError(t, err)
				assert.Equal(t, "ipfs://QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf", ipfsInputSource.Alias)
				assert.Equal(t, "/target", ipfsInputSource.Target)
				assert.Equal(t, storage_ipfs.Source{
					CID: "QmW7TdjNEMzqmWxm5WPK1p6QCkeChxMLpvhLxyUW2wpjCf",
				}, ipfsInput)

				s3InputSource := task.InputSources[1]
				s3Input, err := storage_s3.DecodeSourceSpec(s3InputSource.Source)
				require.NoError(t, err)

				assert.Equal(t, "s3://myBucket/dir/file-001.txt", s3InputSource.Alias)
				assert.Equal(t, "/inputs", s3InputSource.Target)
				assert.Equal(t, storage_s3.SourceSpec{
					Bucket: "myBucket",
					Key:    "dir/file-001.txt",
				}, s3Input)
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
