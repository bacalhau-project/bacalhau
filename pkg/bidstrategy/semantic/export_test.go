//go:build unit || !integration

package semantic_test

import (
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	testing2 "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	testutil "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func getBidStrategyRequest(t testing.TB) bidstrategy.BidStrategyRequest {
	return bidstrategy.BidStrategyRequest{
		NodeID: "node-id",
		Job: model.Job{
			Metadata: model.Metadata{
				ID: "job-id",
			},
			Spec: model.Spec{
				Engine: testing2.NoopMakeEngine(t, "noop"),
			},
		},
	}
}

func getBidStrategyRequestWithInput(t testing.TB) bidstrategy.BidStrategyRequest {
	request := getBidStrategyRequest(t)
	ipfsSpec, err := (&ipfs.IPFSStorageSpec{CID: testutil.TestCID1}).AsSpec("TODO", "TODO")
	if err != nil {
		panic(fmt.Sprintf("developer error: %s", err))
	}
	request.Job.Spec.Inputs = []spec.Storage{ipfsSpec}
	return request
}
