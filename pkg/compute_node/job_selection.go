package compute_node

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

func NewDefaultJobSelectionPolicy() types.JobSelectionPolicy {
	return types.JobSelectionPolicy{
		Data: types.JobSelectionDataPolicy{},
	}
}
