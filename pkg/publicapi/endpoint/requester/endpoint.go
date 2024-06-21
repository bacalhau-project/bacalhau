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
	group.POST("/list", methodGone("https://docs.bacalhau.org/references/api/jobs#list-jobs"))
	group.GET("/nodes", methodGone("https://docs.bacalhau.org/references/api/nodes#list-nodes"))
	group.POST("/states", methodGone("https://docs.bacalhau.org/references/api/jobs#describe-job"))
	group.POST("/results", methodGone("https://docs.bacalhau.org/references/api/jobs#describe-job"))
	group.POST("/events", methodGone("https://docs.bacalhau.org/references/api/jobs#job-history"))
	group.POST("/submit", methodGone("https://docs.bacalhau.org/references/api/jobs#create-job"))
	group.POST("/cancel", methodGone("https://docs.bacalhau.org/references/api/jobs#stop-job"))
	group.POST("/debug", methodGone("https://docs.bacalhau.org/references/api/nodes#describe-node"))
	group.GET("/websocket/events", methodGone("https://docs.bacalhau.org/references/api/jobs#job-history"))
}

const deprecationMessage = "This endpoint is deprecated and no longer available. Please refer to %s for more information. If you encountered this error using the Bacalhau client or CLI, please update your node by following the instructions here: https://docs.bacalhau.org/getting-started/installation" //nolint:lll

func methodGone(docsLink string) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.JSON(http.StatusGone, fmt.Sprintf(deprecationMessage, docsLink))
	}
}
