//go:build unit || !integration

package compute_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/env"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestGetExecutionEnvVars(t *testing.T) {
	resolver := env.NewResolver(env.ResolverParams{
		AllowList: []string{"MY_*", "OTHER_*", "TEST_*"},
	})

	// Set up test environment variables
	t.Setenv("MY_HOST_VAR", "host-value")
	t.Setenv("TEST_VAR", "test-value")

	tests := []struct {
		name      string
		execution *models.Execution
		want      map[string]string
		wantErr   bool
	}{
		{
			name:      "nil execution returns empty map",
			execution: nil,
			want:      map[string]string{},
		},
		{
			name: "basic execution without job",
			execution: &models.Execution{
				ID:     "exec-1",
				NodeID: "node-1",
			},
			want: map[string]string{
				"BACALHAU_EXECUTION_ID": "exec-1",
			},
		},
		{
			name: "execution with job but no task env",
			execution: &models.Execution{
				ID:             "exec-1",
				NodeID:         "node-1",
				JobID:          "job-1",
				PartitionIndex: 0,
				Job: &models.Job{
					Name:      "test-job",
					Namespace: "default",
					Type:      "batch",
					Tasks: []*models.Task{
						{
							Name: "task-1",
						},
					},
					Count: 3,
				},
			},
			want: map[string]string{
				"BACALHAU_EXECUTION_ID":    "exec-1",
				"BACALHAU_JOB_ID":          "job-1",
				"BACALHAU_JOB_TYPE":        "batch",
				"BACALHAU_PARTITION_INDEX": "0",
				"BACALHAU_PARTITION_COUNT": "3",
			},
		},
		{
			name: "execution with literal and resolved env vars",
			execution: &models.Execution{
				ID:             "exec-1",
				NodeID:         "node-1",
				JobID:          "job-1",
				PartitionIndex: 0,
				Job: &models.Job{
					Name:      "test-job",
					Namespace: "default",
					Type:      "batch",
					Tasks: []*models.Task{
						{
							Name: "task-1",
							Env: map[string]models.EnvVarValue{
								"LITERAL_VAR":       "literal-value",
								"HOST_VAR":          "env:MY_HOST_VAR",
								"TEST_VAR":          "env:TEST_VAR",
								"BACALHAU_JOB_TYPE": "should-not-override",
							},
						},
					},
					Count: 3,
				},
			},
			want: map[string]string{
				"BACALHAU_EXECUTION_ID":    "exec-1",
				"BACALHAU_JOB_ID":          "job-1",
				"BACALHAU_JOB_TYPE":        "batch",
				"BACALHAU_PARTITION_INDEX": "0",
				"BACALHAU_PARTITION_COUNT": "3",
				"LITERAL_VAR":              "literal-value",
				"HOST_VAR":                 "host-value",
				"TEST_VAR":                 "test-value",
			},
		},
		{
			name: "execution with not allowed env var",
			execution: &models.Execution{
				ID:             "exec-1",
				NodeID:         "node-1",
				JobID:          "job-1",
				PartitionIndex: 0,
				Job: &models.Job{
					Tasks: []*models.Task{
						{
							Name: "task-1",
							Env: map[string]models.EnvVarValue{
								"NOT_ALLOWED": "env:DENIED_VAR",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "execution with missing env var",
			execution: &models.Execution{
				ID:             "exec-1",
				NodeID:         "node-1",
				JobID:          "job-1",
				PartitionIndex: 0,
				Job: &models.Job{
					Tasks: []*models.Task{
						{
							Name: "task-1",
							Env: map[string]models.EnvVarValue{
								"MISSING": "env:MY_MISSING_VAR",
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compute.GetExecutionEnvVars(tt.execution, resolver)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
