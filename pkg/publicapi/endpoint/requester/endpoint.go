package requester

import (
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type EndpointParams struct {
	Router             *echo.Echo
	Requester          requester.Endpoint
	DebugInfoProviders []model.DebugInfoProvider
	JobStore           jobstore.Store
	NodeDiscoverer     orchestrator.NodeDiscoverer
}

type Endpoint struct {
	router             *echo.Echo
	requester          requester.Endpoint
	debugInfoProviders []model.DebugInfoProvider
	jobStore           jobstore.Store
	nodeDiscoverer     orchestrator.NodeDiscoverer
	// jobId or "" (for all events) -> connections for that subscription
	websockets      map[string][]*websocket.Conn
	websocketsMutex sync.RWMutex
}

func NewEndpoint(params EndpointParams) *Endpoint {
	e := &Endpoint{
		router:             params.Router,
		requester:          params.Requester,
		debugInfoProviders: params.DebugInfoProviders,
		jobStore:           params.JobStore,
		nodeDiscoverer:     params.NodeDiscoverer,
		websockets:         make(map[string][]*websocket.Conn),
	}

	g := e.router.Group("/api/v1/requester")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
	g.POST("/list", e.list)
	g.GET("/nodes", e.nodes)
	g.POST("/states", e.states)
	g.POST("/results", e.results)
	g.POST("/events", e.events)
	g.POST("/submit", e.submit)
	g.POST("/cancel", e.cancel)
	g.POST("/debug", e.debug)
	g.GET("/logs", e.logs)
	g.GET("/websocket/events", e.websocketJobEvents)

	return e
}
