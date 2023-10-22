//go:build unit || !integration

package serve_test

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	apitest "github.com/bacalhau-project/bacalhau/pkg/publicapi/test"
)

var (
	noTimeout          = model.NoJobTimeout
	nonZeroTimeout     = 30 * time.Second
	halfNonZeroTimeout = nonZeroTimeout / 2
)

func (s *ServeSuite) TestNoTimeoutSetOrApplied() {
	docker.MustHaveDocker(s.T())

	cases := []struct {
		configuredMax    *time.Duration
		timeoutSpecified *time.Duration
		timeoutApplied   time.Duration
		stateExpected    model.JobStateType
	}{
		{configuredMax: nil, timeoutSpecified: nil, timeoutApplied: model.NoJobTimeout, stateExpected: model.JobStateCompleted},
		{configuredMax: nil, timeoutSpecified: &nonZeroTimeout, timeoutApplied: nonZeroTimeout, stateExpected: model.JobStateCompleted},
		{configuredMax: &nonZeroTimeout, timeoutSpecified: nil, timeoutApplied: nonZeroTimeout, stateExpected: model.JobStateCompleted},
		{configuredMax: &nonZeroTimeout, timeoutSpecified: &halfNonZeroTimeout, timeoutApplied: halfNonZeroTimeout, stateExpected: model.JobStateCompleted},
		{configuredMax: &nonZeroTimeout, timeoutSpecified: &noTimeout, timeoutApplied: noTimeout, stateExpected: model.JobStateError},
	}

	for _, tc := range cases {
		name := fmt.Sprintf(
			"job in %s has timeout %s after configuring %s and specifying %s",
			tc.stateExpected,
			tc.timeoutApplied,
			tc.configuredMax,
			tc.timeoutSpecified,
		)

		s.Run(name, func() {
			args := []string{}
			if tc.configuredMax != nil {
				args = append(args, "--max-job-execution-timeout", tc.configuredMax.String())
			}

			port, err := s.serve(args...)
			s.Require().NoError(err)

			client := client.NewAPIClient("localhost", port)
			clientV2 := clientv2.New(clientv2.Options{
				Address: fmt.Sprintf("http://127.0.0.1:%d", port),
			})
			s.Require().NoError(apitest.WaitForAlive(s.ctx, clientV2))

			testJob := model.NewJob()
			specOpts := []job.SpecOpt{}
			if tc.timeoutSpecified != nil {
				specOpts = append(specOpts, job.WithTimeout(int64(tc.timeoutSpecified.Seconds())))
			}
			testJob.Spec, err = job.MakeSpec(specOpts...)
			s.Require().NoError(err)

			returnedJob, err := client.Submit(s.ctx, testJob)
			s.Require().NoError(err)

			s.Eventually(func() bool {
				jobState, err := client.GetJobState(s.ctx, returnedJob.ID())
				s.Require().NoError(err)
				s.Require().Equal(model.JobStateError.String(), jobState.State.String())
				return true
			}, 5*time.Second, 50*time.Millisecond)
		})
	}
}
