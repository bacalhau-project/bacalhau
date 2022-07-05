package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/stretchr/testify/assert"
)

func TestEmptySpec(t *testing.T) {
	ctx, span := newSpan("TestEmptySpec")
	defer span.End()

	stack, cm := SetupTest(
		t,
		1,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	_, err := apiClient.Submit(ctx, nil, nil, nil)
	assert.NoError(t, err)
}
