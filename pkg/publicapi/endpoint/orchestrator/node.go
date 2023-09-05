package orchestrator

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/peer"
	"k8s.io/apimachinery/pkg/labels"
)

func (e *Endpoint) getNode(c echo.Context) error {
	ctx := c.Request().Context()
	nodeID, err := peer.Decode(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	job, err := e.nodeStore.Get(ctx, nodeID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, apimodels.GetNodeResponse{
		Node: &job,
	})
}

func (e *Endpoint) listNodes(c echo.Context) error {
	ctx := c.Request().Context()
	var args apimodels.ListNodesRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	selector, err := parseLabels(c)
	if err != nil {
		return err
	}

	// query nodes
	allNodes, err := e.nodeStore.List(ctx)
	if err != nil {
		return err
	}

	// filter nodes
	res := make([]*models.NodeInfo, 0)
	for i, node := range allNodes {
		if selector.Matches(labels.Set(node.Labels)) {
			res = append(res, &allNodes[i])
		}
	}

	return c.JSON(http.StatusOK, &apimodels.ListNodesResponse{
		Nodes: res,
	})
}
