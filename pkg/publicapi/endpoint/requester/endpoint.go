package requester

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
)

type Endpoint struct {
	router *echo.Echo
}

func NewEndpoint(router *echo.Echo) *Endpoint {
	e := &Endpoint{
		router: router,
	}

	g := e.router.Group("/api/v1/requester")
	g.Use(middleware.SetContentType(echo.MIMEApplicationJSON))
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
