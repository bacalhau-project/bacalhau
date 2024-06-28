package requester

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
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

// NB: to whomever removes this, grep for the env var when doing so, we can't
// use this variable everywhere due to circular deps
// TODO: https://github.com/bacalhau-project/bacalhau/issues/4119
const UseDeprecatedEndpointsForTesting = "REQUESTER_ENDPOINT_USE_DEPRECATED_ENV"

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
	if key := os.Getenv(UseDeprecatedEndpointsForTesting); key != "" {
		g.POST("/list", e.list)
		g.GET("/nodes", e.nodes)
		g.POST("/states", e.states)
		g.POST("/results", e.results)
		g.POST("/events", e.events)
		g.POST("/submit", e.submit)
		g.POST("/cancel", e.cancel)
		g.POST("/debug", e.debug)
		g.GET("/websocket/events", e.websocketJobEvents)
		return e
	}

	registerDeprecatedLegacyMethods(g)
	return e
}

// registerDeprecatedLegacyMethods registers routes on the router that are 'Gone'.
func registerDeprecatedLegacyMethods(group *echo.Group) {
	// Legacy API Endpoints
	// All return status 410 https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/410
	group.POST("/list", methodGone())
	group.GET("/nodes", methodGone())
	group.POST("/states", methodGone())
	group.POST("/results", methodGone())
	group.POST("/events", methodGone())
	group.POST("/submit", methodGone())
	group.POST("/cancel", methodGone())
	group.POST("/debug", methodGone())
	group.GET("/websocket/events", methodGone())
}

const MigrationGuideURL = "https://docs.bacalhau.org/references/cli-reference/command-migration"
const deprecationMessage = `This endpoint is deprecated. See the migration guide at %s for more information`

func methodGone() func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.JSON(http.StatusGone, fmt.Sprintf(deprecationMessage, MigrationGuideURL))
	}
}
