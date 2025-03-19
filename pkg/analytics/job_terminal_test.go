package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobTerminalTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestJobTerminalSuite(t *testing.T) {
	suite.Run(t, new(JobTerminalTestSuite))
}

func (s *JobTerminalTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *JobTerminalTestSuite) createTestJob() models.Job {
	return models.Job{
		ID:         "test-job-id",
		Name:       "test-job",
		Namespace:  "test-namespace",
		Type:       "test-type",
		Count:      1,
		Labels:     map[string]string{"key": "value"},
		Meta:       map[string]string{"meta": "value"},
		Version:    1,
		Revision:   1,
		CreateTime: time.Now().UnixNano(),
		ModifyTime: time.Now().UnixNano(),
		Tasks: []*models.Task{
			{
				Name: "test-task",
				Engine: &models.SpecConfig{
					Type: models.EngineDocker,
					Params: map[string]interface{}{
						"Image": "ghcr.io/bacalhau-project/test:latest",
					},
				},
				Publisher: &models.SpecConfig{
					Type: "test-publisher",
				},
				Env: map[string]models.EnvVarValue{
					"ENV": "test",
				},
				Meta: map[string]string{"task-meta": "value"},
				InputSources: []*models.InputSource{
					{Source: &models.SpecConfig{Type: "test-source"}},
				},
				ResultPaths: []*models.ResultPath{
					{Name: "result1"},
					{Name: "result2"},
				},
				Network: &models.NetworkConfig{
					Type:    models.NetworkHTTP,
					Domains: []string{"test.com"},
				},
				Timeouts: &models.TimeoutConfig{
					ExecutionTimeout: 3600,
					QueueTimeout:     1800,
					TotalTimeout:     7200,
				},
			},
		},
	}
}

func (s *JobTerminalTestSuite) TestJobTerminalEvent() {
	job := s.createTestJob()
	job.State = models.NewJobState(models.JobStateTypeCompleted)

	event := NewJobTerminalEvent(job)

	eventData, ok := event.Properties.(JobTerminalEvent)
	s.True(ok, "Properties should be of type JobTerminalEvent")

	s.Equal("test-job-id", eventData.JobID)
	s.True(eventData.NameSet)
	s.Equal(hashString("test-namespace"), eventData.NamespaceHash)
	s.Equal("test-type", eventData.Type)
	s.Equal(1, eventData.Count)
	s.Equal(1, eventData.LabelsCount)
	s.Equal(1, eventData.MetaCount)
	s.Equal(models.JobStateTypeCompleted.String(), eventData.State)
	s.Equal(hashString("test-task"), eventData.TaskNameHash)
	s.Equal(models.EngineDocker, eventData.TaskEngineType)
	s.Equal("test-publisher", eventData.TaskPublisherType)
	s.Equal(1, eventData.TaskEnvVarCount)
	s.Equal(1, eventData.TaskMetaCount)
	s.Equal([]string{"test-source"}, eventData.TaskInputSourceTypes)
	s.Equal(2, eventData.TaskResultPathCount)
	s.Equal("ghcr.io/bacalhau-project/test:latest", eventData.TaskDockerImage)
	s.Equal(models.NetworkHTTP.String(), eventData.TaskNetworkType)
	s.Equal(1, eventData.TaskDomainsCount)
	s.Equal(int64(3600), eventData.TaskExecutionTimeout)
	s.Equal(int64(1800), eventData.TaskQueueTimeout)
	s.Equal(int64(7200), eventData.TaskTotalTimeout)
}
