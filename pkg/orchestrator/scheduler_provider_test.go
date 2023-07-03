//go:build unit || !integration

package orchestrator

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type SchedulerProviderTestSuite struct {
	suite.Suite
	serviceTypeScheduler Scheduler
	batchTypeScheduler   Scheduler
	schedulerProvider    SchedulerProvider
}

func (s *SchedulerProviderTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.serviceTypeScheduler = NewMockScheduler(ctrl)
	s.batchTypeScheduler = NewMockScheduler(ctrl)
	s.schedulerProvider = NewMappedSchedulerProvider(map[string]Scheduler{
		model.JobTypeService: s.serviceTypeScheduler,
		model.JobTypeBatch:   s.batchTypeScheduler,
	})
}

func (s *SchedulerProviderTestSuite) TestGetScheduler() {
	testCases := []struct {
		jobType     string
		expected    Scheduler
		description string
	}{
		{
			jobType:     model.JobTypeService,
			expected:    s.serviceTypeScheduler,
			description: "GetScheduler should return the scheduler for service type",
		},
		{
			jobType:     model.JobTypeBatch,
			expected:    s.batchTypeScheduler,
			description: "GetScheduler should return the scheduler for batch type",
		},
		{
			jobType:     "non_existing_type",
			expected:    nil,
			description: "GetScheduler should return nil for non-existing type",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.description, func() {
			scheduler, err := s.schedulerProvider.Scheduler(tc.jobType)
			s.Equal(tc.expected, scheduler)
			if tc.expected == nil {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *SchedulerProviderTestSuite) TestEnabledSchedulers() {
	s.ElementsMatchf(
		[]string{model.JobTypeService, model.JobTypeBatch},
		s.schedulerProvider.EnabledSchedulers(),
		"EnabledSchedulers should return all the enabled schedulers",
	)
}

func TestSchedulerProviderTestSuite(t *testing.T) {
	suite.Run(t, new(SchedulerProviderTestSuite))
}
