package orchestrator

import (
	"github.com/labstack/echo/v4"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
)

type EndpointParams struct {
	Router       *echo.Echo
	Orchestrator *orchestrator.BaseEndpoint
	JobStore     jobstore.Store
	NodeManager  nodes.Manager
}

type Endpoint struct {
	router       *echo.Echo
	orchestrator *orchestrator.BaseEndpoint
	store        jobstore.Store
	nodeManager  nodes.Manager
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:       params.Router,
		orchestrator: params.Orchestrator,
		store:        params.JobStore,
		nodeManager:  params.NodeManager,
	}

	// JSON group
	g := e.router.Group("/api/v1/orchestrator")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.PUT("/jobs", e.putJob)
	g.POST("/jobs", e.putJob)
	g.GET("/jobs", e.listJobs)
	g.GET("/jobs/:id", e.getJob)
	g.DELETE("/jobs/:id", e.stopJob)
	g.PUT("/jobs/diff", e.diffJob)
	g.PUT("/jobs/:id/rerun", e.rerunJob)
	g.GET("/jobs/:id/history", e.listHistory)
	g.GET("/jobs/:id/executions", e.jobExecutions)
	g.GET("/jobs/:id/versions", e.jobVersions)
	g.GET("/jobs/:id/results", e.jobResults)
	g.GET("/jobs/:id/logs", e.logs)
	g.GET("/nodes", e.listNodes)
	g.GET("/nodes/:id", e.getNode)
	g.PUT("/nodes/:id", e.updateNode)
	return e
}
