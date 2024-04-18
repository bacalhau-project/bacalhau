package orchestrator

import (
	"context"
	"net/http"
	"strings"

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
	nodeState, err := e.nodeManager.GetByPrefix(ctx, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, apimodels.GetNodeResponse{
		Node: &nodeState,
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

	capacity := func(node *models.NodeState) *models.Resources {
		if node.Info.ComputeNodeInfo != nil {
			return &node.Info.ComputeNodeInfo.AvailableCapacity
		}
		return &models.Resources{}
	}

	// parse order_by
	sortFnc := e.getSortFunction(args.OrderBy, capacity)
	if sortFnc == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid order_by")
	}

	if args.Reverse {
		baseSortFnc := sortFnc
		sortFnc = func(a, b *models.NodeState) int {
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

	args.FilterByApproval = strings.ToUpper(args.FilterByApproval)
	args.FilterByStatus = strings.ToUpper(args.FilterByStatus)

	// filter nodes, first by status, then by label selectors
	res := make([]*models.NodeState, 0)
	for i, node := range allNodes {
		if args.FilterByApproval != "" && args.FilterByApproval != node.Membership.String() {
			continue
		}

		if args.FilterByStatus != "" && args.FilterByStatus != node.Connection.String() {
			continue
		}

		if selector.Matches(labels.Set(node.Info.Labels)) {
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

type resourceFunc func(node *models.NodeState) *models.Resources
type sortFunc func(a, b *models.NodeState) int

func (e *Endpoint) getSortFunction(orderBy string, capacity resourceFunc) sortFunc {
	switch orderBy {
	case "id", "":
		return func(a, b *models.NodeState) int { return util.Compare[string]{}.Cmp(a.Info.ID(), b.Info.ID()) }
	case "type":
		return func(a, b *models.NodeState) int {
			return util.Compare[models.NodeType]{}.Cmp(a.Info.NodeType, b.Info.NodeType)
		}
	case "available_cpu":
		return func(a, b *models.NodeState) int {
			return util.Compare[float64]{}.CmpRev(capacity(a).CPU, capacity(b).CPU)
		}
	case "available_memory":
		return func(a, b *models.NodeState) int {
			return util.Compare[uint64]{}.CmpRev(capacity(a).Memory, capacity(b).Memory)
		}
	case "available_disk":
		return func(a, b *models.NodeState) int {
			return util.Compare[uint64]{}.CmpRev(capacity(a).Disk, capacity(b).Disk)
		}
	case "available_gpu":
		return func(a, b *models.NodeState) int {
			return util.Compare[uint64]{}.CmpRev(capacity(a).GPU, capacity(b).GPU)
		}
	case "approval", "status":
		return func(a, b *models.NodeState) int {
			return util.Compare[string]{}.Cmp(a.Membership.String(), b.Membership.String())
		}
	default:
	}

	return nil
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
		action = e.nodeManager.ApproveAction
	} else if args.Action == string(apimodels.NodeActionReject) {
		action = e.nodeManager.RejectAction
	} else if args.Action == string(apimodels.NodeActionDelete) {
		action = e.nodeManager.DeleteAction
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
