//go:build unit || !integration

package compute

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestGetExecutionEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		execution *models.Execution
		want      map[string]string
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
				"BACALHAU_NODE_ID":      "node-1",
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
				"BACALHAU_NODE_ID":         "node-1",
				"BACALHAU_JOB_ID":          "job-1",
				"BACALHAU_JOB_NAME":        "test-job",
				"BACALHAU_JOB_NAMESPACE":   "default",
				"BACALHAU_JOB_TYPE":        "batch",
				"BACALHAU_PARTITION_INDEX": "0",
				"BACALHAU_PARTITION_COUNT": "3",
			},
		},
		{
			name: "execution with task env",
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
							Env: map[string]string{
								"MY_VAR":           "my-value",
								"BACALHAU_NODE_ID": "should-not-override", // Should not override system env
								"OTHER_VAR":        "other-value",
							},
						},
					},
					Count: 3,
				},
			},
			want: map[string]string{
				"BACALHAU_EXECUTION_ID":    "exec-1",
				"BACALHAU_NODE_ID":         "node-1", // System value takes precedence
				"BACALHAU_JOB_ID":          "job-1",
				"BACALHAU_JOB_NAME":        "test-job",
				"BACALHAU_JOB_NAMESPACE":   "default",
				"BACALHAU_JOB_TYPE":        "batch",
				"BACALHAU_PARTITION_INDEX": "0",
				"BACALHAU_PARTITION_COUNT": "3",
				"MY_VAR":                   "my-value",
				"OTHER_VAR":                "other-value",
			},
		},
		{
			name: "execution with special characters in names",
			execution: &models.Execution{
				ID:             "exec-1",
				NodeID:         "node-1",
				JobID:          "job-1",
				PartitionIndex: 0,
				Job: &models.Job{
					Name:      "test=job with spaces",
					Namespace: "test=namespace",
					Type:      "batch",
					Tasks: []*models.Task{
						{
							Name: "task-1",
						},
					},
					Count: 1,
				},
			},
			want: map[string]string{
				"BACALHAU_EXECUTION_ID":    "exec-1",
				"BACALHAU_NODE_ID":         "node-1",
				"BACALHAU_JOB_ID":          "job-1",
				"BACALHAU_JOB_NAME":        "test_job_with_spaces", // Sanitized
				"BACALHAU_JOB_NAMESPACE":   "test_namespace",       // Sanitized
				"BACALHAU_JOB_TYPE":        "batch",
				"BACALHAU_PARTITION_INDEX": "0",
				"BACALHAU_PARTITION_COUNT": "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExecutionEnvVars(tt.execution)
			assert.Equal(t, tt.want, got)
		})
	}
}
