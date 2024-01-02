package orchestrator

import (
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/labstack/echo/v4"
	echo_middleware "github.com/labstack/echo/v4/middleware"
)

type EndpointParams struct {
	Router       *echo.Echo
	Orchestrator *orchestrator.BaseEndpoint
	JobStore     jobstore.Store
	NodeStore    routing.NodeInfoStore
}

type Endpoint struct {
	router       *echo.Echo
	orchestrator *orchestrator.BaseEndpoint
	store        jobstore.Store
	nodeStore    routing.NodeInfoStore
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:       params.Router,
		orchestrator: params.Orchestrator,
		store:        params.JobStore,
		nodeStore:    params.NodeStore,
	}

	// JSON group
	g := e.router.Group("/api/v1/orchestrator")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.Use(echo_middleware.CORS())
	g.PUT("/jobs", e.putJob)
	g.POST("/jobs", e.putJob)
	g.GET("/jobs", e.listJobs)
	g.GET("/jobs/:id", e.getJob)
	g.DELETE("/jobs/:id", e.stopJob)
	g.GET("/jobs/:id/history", e.jobHistory)
	g.GET("/jobs/:id/executions", e.jobExecutions)
	g.GET("/jobs/:id/results", e.jobResults)
	g.GET("/nodes", e.listNodes)
	g.GET("/nodes/:id", e.getNode)
	return e
}
