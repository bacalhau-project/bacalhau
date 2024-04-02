package requester

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// nodes godoc
//
//	@ID			pkg/requester/publicapi/nodes
//	@Summary	Displays the nodes that this requester knows about
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]models.NodeInfo
//	@Failure	500	{object}	string
//	@Router		/api/v1/requester/nodes [get]
func (s *Endpoint) nodes(c echo.Context) error {
	ctx := c.Request().Context()
	nodes, err := s.nodeDiscoverer.ListNodes(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, nodes)
}
