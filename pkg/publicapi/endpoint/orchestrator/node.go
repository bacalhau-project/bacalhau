package orchestrator

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/util"
)

func (e *Endpoint) getNode(c echo.Context) error {
	ctx := c.Request().Context()
	if c.Param("id") == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing node id")
	}
	job, err := e.nodeManager.GetByPrefix(ctx, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, apimodels.GetNodeResponse{
		Node: &job,
	})
}

//nolint:gocyclo // cyclomatic complexity is high here becomes of the complex sorting logic
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
	var sortFnc func(a, b *models.NodeInfo) int
	switch args.OrderBy {
	case "id", "":
		sortFnc = func(a, b *models.NodeInfo) int { return util.Compare[string]{}.Cmp(a.ID(), b.ID()) }
	case "type":
		sortFnc = func(a, b *models.NodeInfo) int { return util.Compare[models.NodeType]{}.Cmp(a.NodeType, b.NodeType) }
	case "available_cpu":
		sortFnc = func(a, b *models.NodeInfo) int {
			return util.Compare[float64]{}.CmpRev(capacity(a).CPU, capacity(b).CPU)
		}
	case "available_memory":
		sortFnc = func(a, b *models.NodeInfo) int {
			return util.Compare[uint64]{}.CmpRev(capacity(a).Memory, capacity(b).Memory)
		}
	case "available_disk":
		sortFnc = func(a, b *models.NodeInfo) int {
			return util.Compare[uint64]{}.CmpRev(capacity(a).Disk, capacity(b).Disk)
		}
	case "available_gpu":
		sortFnc = func(a, b *models.NodeInfo) int {
			return util.Compare[uint64]{}.CmpRev(capacity(a).GPU, capacity(b).GPU)
		}
	case "approval", "status":
		sortFnc = func(a, b *models.NodeInfo) int {
			return util.Compare[string]{}.Cmp(a.Approval.String(), b.Approval.String())
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid order_by")
	}
	if args.Reverse {
		baseSortFnc := sortFnc
		sortFnc = func(a, b *models.NodeInfo) int {
			x := baseSortFnc(a, b)
			if x == -1 {
				return 1
			}
			if x == 1 {
				return -1
			}
			return 0
		}
	}

	// query nodes
	allNodes, err := e.nodeManager.List(ctx)
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

func (e *Endpoint) updateNode(c echo.Context) error {
	ctx := c.Request().Context()

	nodeID := c.Param("id")
	if nodeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing node id")
	}

	var args apimodels.PutNodeRequest
	if err := c.Bind(&args); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&args); err != nil {
		return err
	}

	var action func(context.Context, string, string) (bool, string)
	if args.Action == string(apimodels.NodeActionApprove) {
		action = e.nodeManager.Approve
	} else if args.Action == string(apimodels.NodeActionReject) {
		action = e.nodeManager.Reject
	} else {
		action = func(context.Context, string, string) (bool, string) {
			return false, "unsupported action"
		}
	}

	success, msg := action(ctx, nodeID, args.Message)
	return c.JSON(http.StatusOK, apimodels.PutNodeResponse{
		Success: success,
		Error:   msg,
	})
}
