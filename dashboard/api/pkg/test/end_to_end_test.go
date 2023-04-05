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

type EndToEndSuite struct {
	DashboardTestSuite

	stack  *devstack.DevStack
	host   host.Host
	client *publicapi.RequesterAPIClient
	server *server.DashboardAPIServer

	jobCalled bool
}

func TestEndToEnd(t *testing.T) {
	suite.Run(t, new(EndToEndSuite))
}

func (e2e *EndToEndSuite) SetupSuite() {
	e2e.DashboardTestSuite.SetupSuite()
}

func (e2e *EndToEndSuite) SetupTest() {
	port, err := freeport.GetFreePort()
	e2e.NoError(err)

	e2e.host, err = libp2p.NewHost(port)
	e2e.NoError(err)

	e2e.opts.Libp2pHost = e2e.host
	e2e.T().Cleanup(func() { e2e.host.Close() })

	e2e.DashboardTestSuite.SetupTest()
	e2e.jobCalled = false

	e2e.server, err = server.NewServer(server.ServerOptions{
		Host:      "localhost",
		Port:      12345, // TODO
		JWTSecret: "test",
	}, e2e.api)
	e2e.NoError(err)

	go func() {
		cm := system.NewCleanupManager()
		e2e.T().Cleanup(func() { cm.Cleanup(e2e.ctx) })
		e2e.server.ListenAndServe(e2e.ctx, cm)
	}()

	e2e.T().Setenv("BACALHAU_JOB_APPROVER", system.GetClientID())
	e2e.stack = testutils.SetupTestWithNoopExecutor(
		e2e.ctx,
		e2e.T(),
		devstack.DevStackOptions{NumberOfHybridNodes: 1},
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWith(node.RequesterConfigParams{
			JobSelectionPolicy: model.JobSelectionPolicy{
				ProbeHTTP: e2e.server.URL().JoinPath("/api/v1/jobs/shouldrun").String(),
			},
		}),
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

	e2e.NoError(e2e.api.Start(e2e.ctx))
	e2e.NoError(libp2p.ConnectToPeer(e2e.ctx, e2e.host, e2e.stack.Nodes[0].Host))
}

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
		for _, event := range info.Events {
			e2e.T().Logf("Event %s: %q", event.EventTime, event.EventName)
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

func (e2e *EndToEndSuite) TestCanSeeJob() {
	job, err := model.NewJobWithSaneProductionDefaults()
	job.Spec.Engine = model.EngineNoop
	e2e.NoError(err)

	job, err = e2e.client.Submit(e2e.ctx, job)
	e2e.NoError(err)
	e2e.NotNil(job)

	apiJob := e2e.WaitForJob(job.ID())
	e2e.NotNil(apiJob)
	e2e.Equal(apiJob.Metadata.ID, job.Metadata.ID)
}

func (e2e *EndToEndSuite) TestCanApproveJob() {
	job, err := model.NewJobWithSaneProductionDefaults()
	job.Spec.Engine = model.EngineNoop
	e2e.NoError(err)

	job, err = e2e.client.Submit(e2e.ctx, job)
	e2e.NoError(err)
	e2e.NotNil(job)

	apiJob := e2e.WaitForJob(job.ID())
	e2e.NotNil(apiJob)
	e2e.Empty(apiJob.Status.State.Nodes) // not scheduled yet

	// Request should exist now.
	info, err := e2e.api.GetJobInfo(e2e.ctx, job.ID())
	e2e.NotNil(info)
	e2e.NoError(err)
	e2e.NotEmpty(info.Requests)
	e2e.Equal(job.ID(), info.Requests[0].JobID)
	numEvents := len(info.Events)

	err = e2e.api.ModerateJobWithoutRequest(e2e.ctx, job.ID(), "looks great", true, types.ModerationTypeExecution, e2e.user)
	e2e.NoError(err)

	e2e.WaitForMoreEvents(job.ID(), numEvents)
	e2e.True(e2e.jobCalled)
}

func (e2e *EndToEndSuite) TestCanRejectJob() {
	job, err := model.NewJobWithSaneProductionDefaults()
	job.Spec.Engine = model.EngineNoop
	e2e.NoError(err)

	job, err = e2e.client.Submit(e2e.ctx, job)
	e2e.NoError(err)
	e2e.NotNil(job)

	apiJob := e2e.WaitForJob(job.ID())
	e2e.NotNil(apiJob)
	e2e.Empty(apiJob.Status.State.Nodes) // not scheduled yet

	// Request should exist now.
	info, err := e2e.api.GetJobInfo(e2e.ctx, job.ID())
	e2e.NotNil(info)
	e2e.NoError(err)
	e2e.NotEmpty(info.Requests)
	e2e.Equal(job.ID(), info.Requests[0].JobID)

	err = e2e.api.ModerateJobWithoutRequest(e2e.ctx, job.ID(), "looks bad", false, types.ModerationTypeExecution, e2e.user)
	e2e.NoError(err)

	time.Sleep(time.Second) // can't handle Canceled events in v1beta1 schema.
	e2e.False(e2e.jobCalled)
}
