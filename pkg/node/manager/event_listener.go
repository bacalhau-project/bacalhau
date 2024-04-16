package manager

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

type NodeEventListener struct {
	broker   orchestrator.EvaluationBroker
	jobstore jobstore.Store
}

func NewNodeEventListener(broker orchestrator.EvaluationBroker, jobstore jobstore.Store) *NodeEventListener {
	return &NodeEventListener{
		broker:   broker,
		jobstore: jobstore,
	}
}

// HandleNodeEvent will receive events from the node manager, and is responsible for deciding what
// to do in response to those events.  This NodeEventHandler implementation is expected to
// create new evaluations based on the events received.
func (n *NodeEventListener) HandleNodeEvent(ctx context.Context, info models.NodeInfo, evt NodeEvent) {
	log.Ctx(ctx).Info().Msgf("Received node event %s for node %s", evt.String(), info.NodeID)
}

var _ NodeEventHandler = &NodeEventListener{}
