//go:build unit || !integration

package serve_test

import (
	"fmt"
	"math"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	apitest "github.com/bacalhau-project/bacalhau/pkg/publicapi/test"
)

var (
	noTimeout          = time.Duration(math.MaxInt64).Truncate(time.Second)
	nonZeroTimeout     = 30 * time.Second
	halfNonZeroTimeout = nonZeroTimeout / 2
)

func (s *ServeSuite) TestNoTimeoutSetOrApplied() {
	docker.MustHaveDocker(s.T())

	cases := []struct {
		configuredMax    *time.Duration
		timeoutSpecified *time.Duration
		stateExpected    models.JobStateType
	}{
		{configuredMax: nil, timeoutSpecified: nil, stateExpected: models.JobStateTypeCompleted},
		{configuredMax: nil, timeoutSpecified: &nonZeroTimeout, stateExpected: models.JobStateTypeCompleted},
		{configuredMax: &nonZeroTimeout, timeoutSpecified: nil, stateExpected: models.JobStateTypeCompleted},
		{configuredMax: &nonZeroTimeout, timeoutSpecified: &halfNonZeroTimeout, stateExpected: models.JobStateTypeCompleted},
		{configuredMax: &nonZeroTimeout, timeoutSpecified: &noTimeout, stateExpected: models.JobStateTypeFailed},
	}

	for _, tc := range cases {
		name := fmt.Sprintf(
			"job in %s configuring timeout %s and specifying %s",
			tc.stateExpected,
			tc.configuredMax,
			tc.timeoutSpecified,
		)

		s.Run(name, func() {
			args := []string{"--node-type", "requester,compute"}
			if tc.configuredMax != nil {
				args = append(args, "--max-job-execution-timeout", tc.configuredMax.String())
			}

			port, err := s.serve(args...)
			s.Require().NoError(err)

			clientV2 := clientv2.New(fmt.Sprintf("http://127.0.0.1:%d", port))
			s.Require().NoError(apitest.WaitForAlive(s.ctx, clientV2))

			testJob := &models.Job{
				Name:  s.T().Name(),
				Type:  models.JobTypeBatch,
				Count: 1,
				Tasks: []*models.Task{
					{
						Name: s.T().Name(),
						Engine: &models.SpecConfig{
							Type:   models.EngineNoop,
							Params: make(map[string]interface{}),
						},
						Publisher: &models.SpecConfig{
							Type:   models.PublisherNoop,
							Params: make(map[string]interface{}),
						},
					},
				},
			}

			if tc.timeoutSpecified != nil {
				testJob.Task().Timeouts = &models.TimeoutConfig{
					TotalTimeout: int64(tc.timeoutSpecified.Seconds()),
				}
			}

			putResp, err := clientV2.Jobs().Put(s.ctx, &apimodels.PutJobRequest{
				Job: testJob,
			})
			s.Require().NoError(err)

			s.Eventually(func() bool {
				getResp, err := clientV2.Jobs().Get(s.ctx, &apimodels.GetJobRequest{JobID: putResp.JobID})
				s.Require().NoError(err)
				s.Require().Equal(models.JobStateTypeFailed, getResp.Job.State.StateType)
				return true
			}, 5*time.Second, 50*time.Millisecond)
		})
	}
}
