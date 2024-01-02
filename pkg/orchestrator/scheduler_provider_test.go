//go:build unit || !integration

package orchestrator

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
		models.JobTypeService: s.serviceTypeScheduler,
		models.JobTypeBatch:   s.batchTypeScheduler,
	})
}

func (s *SchedulerProviderTestSuite) TestGetScheduler() {
	testCases := []struct {
		jobType     string
		expected    Scheduler
		description string
	}{
		{
			jobType:     models.JobTypeService,
			expected:    s.serviceTypeScheduler,
			description: "GetScheduler should return the scheduler for service type",
		},
		{
			jobType:     models.JobTypeBatch,
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
		[]string{models.JobTypeService, models.JobTypeBatch},
		s.schedulerProvider.EnabledSchedulers(),
		"EnabledSchedulers should return all the enabled schedulers",
	)
}

func TestSchedulerProviderTestSuite(t *testing.T) {
	suite.Run(t, new(SchedulerProviderTestSuite))
}
