// File: pkg/compute/envvars.go

package compute

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envvar"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ResolveEnvVars resolves all environment variables in the given map using the provided resolver
func ResolveEnvVars(resolver EnvVarResolver, env map[string]models.EnvVarValue) (map[string]string, error) {
	resolved := make(map[string]string)

	for name, value := range env {
		val, err := resolver.Value(string(value))
		if err != nil {
			return nil, err
		}
		resolved[name] = val
	}

	return resolved, nil
}

// GetExecutionEnvVars returns a map of environment variables that should be passed to the execution.
// Task-level environment variables are included but system variables take precedence.
func GetExecutionEnvVars(execution *models.Execution, resolver EnvVarResolver) (map[string]string, error) {
	if execution == nil {
		return make(map[string]string), nil
	}

	// Start with task-level environment variables if they exist
	taskEnv := make(map[string]string)
	if execution.Job != nil && execution.Job.Task() != nil && execution.Job.Task().Env != nil {
		resolved, err := ResolveEnvVars(resolver, execution.Job.Task().Env)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve task environment variables: %w", err)
		}
		taskEnv = resolved
	}

	// Build system environment variables
	sysEnv := make(map[string]string)
	sysEnv[models.EnvVarPrefix+"EXECUTION_ID"] = execution.ID

	// Add job-related environment variables if job exists
	if execution.Job != nil {
		sysEnv[models.EnvVarPrefix+"JOB_ID"] = execution.JobID
		sysEnv[models.EnvVarPrefix+"JOB_TYPE"] = execution.Job.Type

		// Add partition-related environment variables
		sysEnv[models.EnvVarPrefix+"PARTITION_INDEX"] = fmt.Sprintf("%d", execution.PartitionIndex)
		sysEnv[models.EnvVarPrefix+"PARTITION_COUNT"] = fmt.Sprintf("%d", execution.Job.Count)

		// Add port-related environment variables
		if execution.Job.Task().Network != nil {
			for _, port := range execution.Job.Task().Network.Ports {
				if port.Static > 0 {
					sysEnv[models.EnvVarHostPortPrefix+port.Name] = fmt.Sprintf("%d", port.Static)
				}
				if port.Target > 0 {
					sysEnv[models.EnvVarPortPrefix+port.Name] = fmt.Sprintf("%d", port.Target)
				}
			}
		}
	}

	// Merge task and system variables, with system taking precedence
	return envvar.Merge(taskEnv, sysEnv), nil
}
