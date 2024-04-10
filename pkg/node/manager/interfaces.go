//go:generate mockgen --source interfaces.go --destination mocks.go --package manager
package manager

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type NodeEventHandler interface {
	HandleNodeEvent(ctx context.Context, info models.NodeInfo, event NodeEvent)
}
