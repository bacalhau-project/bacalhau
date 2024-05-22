package helpers

import (
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

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
	t, err := models.NewTaskBuilder().
		Name(taskSettings.Name).
		Engine(engineSpec).
		Publisher(taskSettings.Publisher).
		ResourcesConfig(&models.ResourcesConfig{
			CPU:    taskSettings.Resources.CPU,
			Memory: taskSettings.Resources.Memory,
			Disk:   taskSettings.Resources.Disk,
			GPU:    taskSettings.Resources.GPU,
		}).
		InputSources(taskSettings.InputSources...).
		ResultPaths(taskSettings.ResultPaths...).
		Network(&models.NetworkConfig{
			Type:    taskSettings.Network.Network,
			Domains: taskSettings.Network.Domains,
		}).
		Timeouts(&models.TimeoutConfig{ExecutionTimeout: taskSettings.Timeout}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	constraints, err := jobSettings.Constraints()
	if err != nil {
		return nil, fmt.Errorf("failed to parse job constraints: %w", err)
	}

	job := &models.Job{
		Name:        jobSettings.Name(),
		Namespace:   jobSettings.Namespace(),
		Type:        jobSettings.Type(),
		Priority:    jobSettings.Priority(),
		Count:       jobSettings.Count(),
		Constraints: constraints,
		Labels:      jobSettings.Labels(),
		Tasks:       []*models.Task{t},
	}

	return job, nil
}
