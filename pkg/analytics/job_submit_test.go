package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobSubmitTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestJobSubmitSuite(t *testing.T) {
	suite.Run(t, new(JobSubmitTestSuite))
}

func (s *JobSubmitTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *JobSubmitTestSuite) createTestJob() models.Job {
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

func (s *JobSubmitTestSuite) TestSubmitJobEvent() {
	job := s.createTestJob()
	event := NewSubmitJobEvent(job)

	s.Equal("test-job-id", event.JobID)
	s.True(event.NameSet)
	s.Equal(hashString("test-namespace"), event.NamespaceHash)
	s.Equal("test-type", event.Type)
	s.Equal(1, event.Count)
	s.Equal(1, event.LabelsCount)
	s.Equal(1, event.MetaCount)
	s.Equal(hashString("test-task"), event.TaskNameHash)
	s.Equal(models.EngineDocker, event.TaskEngineType)
	s.Equal("test-publisher", event.TaskPublisherType)
	s.Equal(1, event.TaskEnvVarCount)
	s.Equal(1, event.TaskMetaCount)
	s.Equal([]string{"test-source"}, event.TaskInputSourceTypes)
	s.Equal(2, event.TaskResultPathCount)
	s.Equal("ghcr.io/bacalhau-project/test:latest", event.TaskDockerImage)
	s.Equal(models.NetworkHTTP.String(), event.TaskNetworkType)
	s.Equal(1, event.TaskDomainsCount)
	s.Equal(int64(3600), event.TaskExecutionTimeout)
	s.Equal(int64(1800), event.TaskQueueTimeout)
	s.Equal(int64(7200), event.TaskTotalTimeout)
}
