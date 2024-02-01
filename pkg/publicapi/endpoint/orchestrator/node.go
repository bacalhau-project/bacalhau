package orchestrator

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/labels"
)

func (e *Endpoint) getNode(c echo.Context) error {
	ctx := c.Request().Context()
	if c.Param("id") == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing node id")
	}
	job, err := e.nodeStore.GetByPrefix(ctx, c.Param("id"))
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

	// parse labels
	selector, err := parseLabels(c)
	if err != nil {
		return err
	}

	capacity := func(node *models.NodeInfo) *models.Resources {
		if node.ComputeNodeInfo != nil {
			return &node.ComputeNodeInfo.AvailableCapacity
		}
		return &models.Resources{}
	}

	// parse order_by
	var sortFnc func(a, b *models.NodeInfo) bool
	switch args.OrderBy {
	case "id", "":
		sortFnc = func(a, b *models.NodeInfo) bool { return a.ID() < b.ID() }
	case "type":
		sortFnc = func(a, b *models.NodeInfo) bool { return a.NodeType < b.NodeType }
	case "available_cpu":
		sortFnc = func(a, b *models.NodeInfo) bool { return capacity(a).CPU > capacity(b).CPU }
	case "available_memory":
		sortFnc = func(a, b *models.NodeInfo) bool { return capacity(a).Memory > capacity(b).Memory }
	case "available_disk":
		sortFnc = func(a, b *models.NodeInfo) bool { return capacity(a).Disk > capacity(b).Disk }
	case "available_gpu":
		sortFnc = func(a, b *models.NodeInfo) bool { return capacity(a).GPU > capacity(b).GPU }
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid order_by")
	}
	if args.Reverse {
		baseSortFnc := sortFnc
		sortFnc = func(a, b *models.NodeInfo) bool { return !baseSortFnc(a, b) }
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

	// sort nodes
	slices.SortFunc(res, sortFnc)

	// apply limit
	// TODO: return next_token for pagination
	if args.Limit > 0 && len(res) > int(args.Limit) {
		res = res[:args.Limit]
	}

	return c.JSON(http.StatusOK, &apimodels.ListNodesResponse{
		Nodes: res,
	})
}
