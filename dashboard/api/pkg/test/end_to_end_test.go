//go:build integration || !unit

package test

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/server"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
)

func ifNil[T comparable](value, defaultValue T) T {
	var zero T
	if value == zero {
		return defaultValue
	} else {
		return value
	}
}

type EndToEndSuite struct {
	DashboardTestSuite

	getRequesterConfig *func(*EndToEndSuite) node.RequesterConfig
	getComputeConfig   *func(*EndToEndSuite) node.ComputeConfig

	stack  *devstack.DevStack
	host   host.Host
	client *publicapi.RequesterAPIClient
	server *server.DashboardAPIServer

	jobCalled bool
}

func (e2e *EndToEndSuite) SetupTest() {
	port, err := freeport.GetFreePort()
	e2e.Require().NoError(err)

	e2e.host, err = libp2p.NewHost(port)
	e2e.Require().NoError(err)

	e2e.opts.Libp2pHost = e2e.host
	e2e.T().Cleanup(func() { e2e.host.Close() })

	e2e.DashboardTestSuite.SetupTest()
	e2e.jobCalled = false

	e2e.server, err = server.NewServer(server.ServerOptions{
		Host:      "localhost",
		Port:      12345, // TODO
		JWTSecret: "test",
	}, e2e.api)
	e2e.Require().NoError(err)

	go func() {
		cm := system.NewCleanupManager()
		e2e.T().Cleanup(func() { cm.Cleanup(e2e.ctx) })
		e2e.server.ListenAndServe(e2e.ctx, cm)
	}()

	getComputeConfig := ifNil(e2e.getComputeConfig, &defaultComputeConfig)
	e2e.Require().NotNil(getComputeConfig)
	getRequesterConfig := ifNil(e2e.getRequesterConfig, &defaultRequesterConfig)
	e2e.Require().NotNil(getRequesterConfig)

	e2e.T().Setenv("BACALHAU_JOB_APPROVER", system.GetClientID())
	e2e.stack = testutils.SetupTestWithNoopExecutor(
		e2e.ctx,
		e2e.T(),
		devstack.DevStackOptions{NumberOfHybridNodes: 1},
		(*getComputeConfig)(e2e),
		(*getRequesterConfig)(e2e),
		noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, job model.Job, resultsDir string) (*model.RunCommandResult, error) {
					e2e.jobCalled = true
					return executor.WriteJobResults(resultsDir, nil, nil, 0, nil)
				},
			},
		},
	)

	node := e2e.stack.Nodes[0]
	e2e.T().Cleanup(func() {
		node.CleanupManager.Cleanup(context.Background())
	})
	e2e.client = publicapi.NewRequesterAPIClient(node.APIServer.Address, node.APIServer.Port)

	e2e.Require().NoError(e2e.api.Start(e2e.ctx))
	e2e.Require().NoError(libp2p.ConnectToPeer(e2e.ctx, e2e.host, e2e.stack.Nodes[0].Host))
}

var (
	defaultComputeConfig   = func(*EndToEndSuite) node.ComputeConfig { return node.NewComputeConfigWithDefaults() }
	defaultRequesterConfig = func(*EndToEndSuite) node.RequesterConfig { return node.NewRequesterConfigWithDefaults() }
)

func (e2e *EndToEndSuite) TearDownTest() {
	e2e.NoError(e2e.api.Stop(context.Background()))
	e2e.DashboardTestSuite.TearDownTest()
}

func (e2e *EndToEndSuite) WaitForJob(id string) (apiJob *v1beta1.Job) {
	for {
		var err error
		apiJob, err = e2e.api.GetJob(e2e.ctx, id)
		if err == nil {
			break
		}
		if err = e2e.ctx.Err(); err != nil {
			e2e.FailNow(err.Error())
		}
		e2e.T().Logf("Did not see job %q after %s", id, SpinUpWaitTime.String())
		time.Sleep(SpinUpWaitTime)
	}
	return
}

func (e2e *EndToEndSuite) WaitForMoreEvents(id string, numEvents int) {
	for {
		info, err := e2e.api.GetJobInfo(e2e.ctx, id)
		for i, event := range info.Events {
			e2e.T().Logf("Event %d: %q", i, event.EventName)
		}
		if cerr := e2e.ctx.Err(); cerr != nil {
			e2e.FailNow(cerr.Error(), "waiting for more events")
		} else if info == nil || err != nil || len(info.Events) == numEvents {
			time.Sleep(SpinUpWaitTime)
			continue
		}
		break
	}
}

type PreModerationTestSuite struct {
	EndToEndSuite
}

func getModeratePolicy(pre *EndToEndSuite) model.JobSelectionPolicy {
	return model.JobSelectionPolicy{
		ProbeHTTP: pre.server.URL().JoinPath("/api/v1/jobs/shouldrun").String(),
	}
}

var (
	preModerateComputeConfig = func(pre *EndToEndSuite) node.ComputeConfig {
		return node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: getModeratePolicy(pre),
		})
	}

	preModerateRequesterConfig = func(pre *EndToEndSuite) node.RequesterConfig {
		return node.NewRequesterConfigWith(node.RequesterConfigParams{
			JobSelectionPolicy: getModeratePolicy(pre),
		})
	}
)

// Run tests where it is the requester node that has the moderation hook
// and will call out to the dashboard for moderation.
func TestPreModerationWithRequesterNodeHook(t *testing.T) {
	s := PreModerationTestSuite{EndToEndSuite: EndToEndSuite{getRequesterConfig: &preModerateRequesterConfig}}
	suite.Run(t, &s)
}

// Run tests where it is the compute node that has the moderation hook
// and will call out to the dashboard for moderation.
func TestPreModerationWithComputeNodeHook(t *testing.T) {
	s := PreModerationTestSuite{EndToEndSuite: EndToEndSuite{getComputeConfig: &preModerateComputeConfig}}
	suite.Run(t, &s)
}

// Run tests where both nodes that have the moderation hook.
// Requester will call out and wait, then compute node will check.
func TestPreModerationWithBothHooks(t *testing.T) {
	s := PreModerationTestSuite{EndToEndSuite: EndToEndSuite{getRequesterConfig: &preModerateRequesterConfig, getComputeConfig: &preModerateComputeConfig}}
	suite.Run(t, &s)
}

func (e2e *PreModerationTestSuite) TestCanSeeJob() {
	job, err := model.NewJobWithSaneProductionDefaults()
	job.Spec.Engine = model.EngineNoop
	e2e.Require().NoError(err)

	job, err = e2e.client.Submit(e2e.ctx, job)
	e2e.Require().NoError(err)
	e2e.Require().NotNil(job)

	apiJob := e2e.WaitForJob(job.ID())
	e2e.Require().NotNil(apiJob)
	e2e.Require().Equal(apiJob.Metadata.ID, job.Metadata.ID)
}

func (e2e *PreModerationTestSuite) TestCanApproveJob() {
	job, err := model.NewJobWithSaneProductionDefaults()
	job.Spec.Engine = model.EngineNoop
	e2e.Require().NoError(err)

	job, err = e2e.client.Submit(e2e.ctx, job)
	e2e.Require().NoError(err)
	e2e.Require().NotNil(job)

	apiJob := e2e.WaitForJob(job.ID())
	e2e.Require().NotNil(apiJob)
	if e2e.getRequesterConfig != nil {
		// The requester is awaiting moderation, so no compute nodes should know
		// about this yet.
		e2e.Require().Empty(apiJob.Status.State.Nodes)
	} else if e2e.getComputeConfig != nil {
		// The compute node is awaiting moderation, so we should be in a
		// non-execution state.
		exes, err := e2e.stack.Nodes[0].ComputeNode.ExecutionStore.GetExecutions(e2e.ctx, job.ID())
		e2e.Require().NoError(err)
		for _, execution := range exes {
			e2e.Require().False(execution.State.IsExecuting())
		}
	}

	// Request should exist now.
	info, err := e2e.api.GetJobInfo(e2e.ctx, job.ID())
	e2e.Require().NotNil(info)
	e2e.Require().NoError(err)
	e2e.Require().NotEmpty(info.Requests)

	req, ok := info.Requests[0].(*types.JobModerationRequest)
	e2e.Require().True(ok)
	e2e.Require().Equal(job.ID(), req.JobID)
	numEvents := len(info.Events)

	err = e2e.api.ModerateJob(e2e.ctx, req.GetID(), "looks great", true, e2e.user)
	e2e.Require().NoError(err)

	e2e.WaitForMoreEvents(job.ID(), numEvents)
	e2e.Require().True(e2e.jobCalled)
}

func (e2e *PreModerationTestSuite) TestCanRejectJob() {
	job, err := model.NewJobWithSaneProductionDefaults()
	job.Spec.Engine = model.EngineNoop
	e2e.Require().NoError(err)

	job, err = e2e.client.Submit(e2e.ctx, job)
	e2e.Require().NoError(err)
	e2e.Require().NotNil(job)

	apiJob := e2e.WaitForJob(job.ID())
	e2e.Require().NotNil(apiJob)
	if e2e.getRequesterConfig != nil {
		// The requester is awaiting moderation, so no compute nodes should know
		// about this yet.
		e2e.Require().Empty(apiJob.Status.State.Nodes)
	} else if e2e.getComputeConfig != nil {
		// The compute node is awaiting moderation, so we should be in a
		// non-execution state.
		exes, err := e2e.stack.Nodes[0].ComputeNode.ExecutionStore.GetExecutions(e2e.ctx, job.ID())
		e2e.Require().NoError(err)
		e2e.Require().NotEmpty(exes)
		for _, execution := range exes {
			e2e.Require().False(execution.State.IsExecuting())
		}
	}

	// Request should exist now.
	info, err := e2e.api.GetJobInfo(e2e.ctx, job.ID())
	e2e.Require().NotNil(info)
	e2e.Require().NoError(err)
	e2e.Require().NotEmpty(info.Requests)
	req, ok := info.Requests[0].(*types.JobModerationRequest)
	e2e.Require().True(ok)
	e2e.Require().Equal(job.ID(), req.JobID)

	err = e2e.api.ModerateJob(e2e.ctx, req.GetID(), "looks bad", false, e2e.user)
	e2e.Require().NoError(err)

	time.Sleep(time.Second) // can't handle Canceled events in v1beta1 schema.
	e2e.Require().False(e2e.jobCalled)
}

type PostModerationTestSuite struct {
	EndToEndSuite
}

var getPostModerationConfig = func(post *EndToEndSuite) node.RequesterConfig {
	return node.NewRequesterConfigWith(node.RequesterConfigParams{
		ExternalValidatorWebhook: post.server.URL().JoinPath("/api/v1/jobs/shouldverify"),
	})
}

func TestPostModeration(t *testing.T) {
	suite.Run(t, &PostModerationTestSuite{EndToEndSuite: EndToEndSuite{getRequesterConfig: &getPostModerationConfig}})
}

func (e2e *PostModerationTestSuite) TestCanApproveJob() {
	j, err := model.NewJobWithSaneProductionDefaults()
	j.Spec.Engine = model.EngineNoop
	j.Spec.Verifier = model.VerifierExternal
	e2e.Require().NoError(err)

	j, err = e2e.client.Submit(e2e.ctx, j)
	e2e.Require().NoError(err)
	e2e.Require().NotNil(j)

	apiJob := e2e.WaitForJob(j.ID())
	e2e.Require().NotNil(apiJob)

	err = e2e.client.GetJobStateResolver().Wait(e2e.ctx, j.ID(),
		job.WaitForExecutionStates(map[model.ExecutionStateType]int{model.ExecutionStateResultProposed: 1}),
		job.WaitDontExceedCount(1),
	)
	e2e.Require().NoError(err)
	e2e.Require().True(e2e.jobCalled)

	// Request should exist now.
	info, err := e2e.api.GetJobInfo(e2e.ctx, j.ID())
	e2e.Require().NotNil(info)
	e2e.Require().NoError(err)
	e2e.Require().NotEmpty(info.Requests)
	req, ok := info.Requests[0].(*types.ResultModerationRequest)
	e2e.Require().True(ok)
	e2e.Require().Equal(j.ID(), req.JobID)

	err = e2e.api.ModerateJob(e2e.ctx, req.GetID(), "looks good", true, e2e.user)
	e2e.Require().NoError(err)

	err = e2e.client.GetJobStateResolver().WaitUntilComplete(e2e.ctx, j.ID())
	e2e.Require().NoError(err)
}

func (e2e *PostModerationTestSuite) TestCanRejectJob() {
	j, err := model.NewJobWithSaneProductionDefaults()
	j.Spec.Engine = model.EngineNoop
	j.Spec.Verifier = model.VerifierExternal
	e2e.Require().NoError(err)

	j, err = e2e.client.Submit(e2e.ctx, j)
	e2e.Require().NoError(err)
	e2e.Require().NotNil(j)

	apiJob := e2e.WaitForJob(j.ID())
	e2e.Require().NotNil(apiJob)

	err = e2e.client.GetJobStateResolver().Wait(e2e.ctx, j.ID(),
		job.WaitForExecutionStates(map[model.ExecutionStateType]int{model.ExecutionStateResultProposed: 1}),
		job.WaitDontExceedCount(1),
	)
	e2e.Require().NoError(err)
	e2e.Require().True(e2e.jobCalled)

	// Request should exist now.
	info, err := e2e.api.GetJobInfo(e2e.ctx, j.ID())
	e2e.Require().NotNil(info)
	e2e.Require().NoError(err)
	e2e.Require().NotEmpty(info.Requests)
	req, ok := info.Requests[0].(*types.ResultModerationRequest)
	e2e.Require().True(ok)
	e2e.Require().Equal(j.ID(), req.JobID)

	err = e2e.api.ModerateJob(e2e.ctx, req.GetID(), "looks bad", false, e2e.user)
	e2e.Require().NoError(err)

	err = e2e.client.GetJobStateResolver().Wait(e2e.ctx, j.ID(),
		job.WaitForExecutionStates(map[model.ExecutionStateType]int{model.ExecutionStateResultRejected: 1}),
		job.WaitDontExceedCount(1),
	)
	e2e.Require().NoError(err)
}
