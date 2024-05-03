package helpers

import (
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// TODO(forrest) [refactor]: this options is _almost_ useful. At present it marshals the entire
// job spec to yaml, said spec cannot be used with `bacalhau job run` since it contains fields that
// users are not permitted to set, like ID, Version, ModifyTime, State, etc.
// The solution here is to have a "JobSubmission" type that is different from the actual job spec.
func JobToYaml(job *models.Job) (string, error) {
	yamlBytes, err := yaml.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("converting job to yaml: %w", err)
	}
	return string(yamlBytes), nil
}

func PrintWarnings(cmd *cobra.Command, warnings []string) {
	cmd.Println("Warnings:")
	for _, warning := range warnings {
		cmd.Printf("\t* %s\n", warning)
	}
}

func BuildJobFromFlags(
	engineSpec *models.SpecConfig,
	jobSettings *cliflags.JobSettings,
	taskSettings *cliflags.TaskSettings,
) (*models.Job, error) {
	resultPaths := make([]*models.ResultPath, 0, len(taskSettings.ResultPaths))
	for name, path := range taskSettings.ResultPaths {
		resultPaths = append(resultPaths, &models.ResultPath{
			Name: name,
			Path: path,
		})
	}

	t, err := models.NewTaskBuilder().
		Name(taskSettings.Name).
		Engine(engineSpec).
		Publisher(taskSettings.Publisher.Value()).
		ResourcesConfig(&models.ResourcesConfig{
			CPU:    taskSettings.Resources.CPU,
			Memory: taskSettings.Resources.Memory,
			Disk:   taskSettings.Resources.Disk,
			GPU:    taskSettings.Resources.GPU,
		}).
		InputSources(taskSettings.InputSources.Values()...).
		ResultPaths(resultPaths...).
		Network(&models.NetworkConfig{
			Type:    taskSettings.Network.Network,
			Domains: taskSettings.Network.Domains,
		}).
		Timeouts(&models.TimeoutConfig{ExecutionTimeout: taskSettings.Timeout}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	constraints, err := parse.NodeSelector(jobSettings.Constraints)
	if err != nil {
		return nil, fmt.Errorf("parseing job contstrints: %w", err)
	}

	job := &models.Job{
		Name:        jobSettings.Name,
		Namespace:   jobSettings.Namespace,
		Type:        jobSettings.Type,
		Priority:    jobSettings.Priority,
		Count:       jobSettings.Count,
		Constraints: constraints,
		Labels:      jobSettings.Labels,
		Tasks:       []*models.Task{t},
	}

	return job, nil
}
