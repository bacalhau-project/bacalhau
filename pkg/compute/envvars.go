// File: pkg/compute/envvars.go

package compute

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envvar"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// GetExecutionEnvVars returns a map of environment variables that should be passed to the execution.
// Task-level environment variables are included but system variables take precedence.
func GetExecutionEnvVars(execution *models.Execution) map[string]string {
	if execution == nil {
		return make(map[string]string)
	}

	// Start with task-level environment variables if they exist
	taskEnv := make(map[string]string)
	if execution.Job != nil && execution.Job.Task() != nil && execution.Job.Task().Env != nil {
		taskEnv = execution.Job.Task().Env
	}

	// Build system environment variables
	sysEnv := make(map[string]string)
	sysEnv[models.EnvVarPrefix+"EXECUTION_ID"] = execution.ID
	sysEnv[models.EnvVarPrefix+"NODE_ID"] = envvar.Sanitize(execution.NodeID)

	// Add job-related environment variables if job exists
	if execution.Job != nil {
		sysEnv[models.EnvVarPrefix+"JOB_ID"] = execution.JobID
		sysEnv[models.EnvVarPrefix+"JOB_NAME"] = envvar.Sanitize(execution.Job.Name)
		sysEnv[models.EnvVarPrefix+"JOB_NAMESPACE"] = envvar.Sanitize(execution.Job.Namespace)
		sysEnv[models.EnvVarPrefix+"JOB_TYPE"] = execution.Job.Type

		// Add partition-related environment variables
		sysEnv[models.EnvVarPrefix+"PARTITION_INDEX"] = fmt.Sprintf("%d", execution.PartitionIndex)
		sysEnv[models.EnvVarPrefix+"PARTITION_COUNT"] = fmt.Sprintf("%d", execution.Job.Count)
	}

	// Merge task and system variables, with system taking precedence
	return envvar.Merge(taskEnv, sysEnv)
}
