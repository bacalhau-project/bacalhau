package helpers

import (
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
)

func JobToYaml(job *models.Job) (string, error) {
	yamlBytes, err := yaml.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("converting job to yaml: %w", err)
	}
	return string(yamlBytes), nil
}

func BuildJobFromFlags(
	engineSpec *models.SpecConfig,
	jobSettings *cliflags.JobSettings,
	taskSettings *cliflags.TaskSettings,
) (*models.Job, error) {
	task := &models.Task{
		Name:      taskSettings.Name,
		Engine:    engineSpec,
		Publisher: taskSettings.Publisher.Value(),
		ResourcesConfig: &models.ResourcesConfig{
			CPU:    taskSettings.Resources.CPU,
			Memory: taskSettings.Resources.Memory,
			Disk:   taskSettings.Resources.Disk,
			GPU:    taskSettings.Resources.GPU,
		},
		InputSources: taskSettings.InputSources.Values(),
		ResultPaths:  taskSettings.ResultPaths,
		Network: &models.NetworkConfig{
			Type:    taskSettings.Network.Network,
			Domains: taskSettings.Network.Domains,
		},
		Timeouts: &models.TimeoutConfig{
			TotalTimeout: taskSettings.Timeout,
			QueueTimeout: taskSettings.QueueTimeout,
		},
		Env: models.EnvVarsFromStringsMap(taskSettings.EnvironmentVariables),
	}
	constraints, err := jobSettings.Constraints()
	if err != nil {
		return nil, fmt.Errorf("failed to parse job constraints: %w", err)
	}

	labels, err := jobSettings.Labels()
	if err != nil {
		return nil, fmt.Errorf("received invalid job labels: %w", err)
	}
	job := &models.Job{
		Name:        jobSettings.Name(),
		Namespace:   jobSettings.Namespace(),
		Type:        jobSettings.Type(),
		Priority:    jobSettings.Priority(),
		Count:       jobSettings.Count(),
		Constraints: constraints,
		Labels:      labels,
		Tasks:       []*models.Task{task},
	}

	// Normalize and validate the job spec
	job.Normalize()
	if err := job.ValidateSubmission(); err != nil {
		return nil, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	return job, nil
}
