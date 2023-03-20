//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"testing"

	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/suite"
)

type WasmRunSuite struct {
	BaseSuite
}

func TestWasmRunSuite(t *testing.T) {
	Fatal = FakeFatalErrorHandler
	suite.Run(t, new(WasmRunSuite))
}

func (s *WasmRunSuite) Test_SupportsRelativeDirectory() {
	ctx := context.Background()
	_, out, err := ExecuteTestCobraCommand("wasm", "run",
		"--api-host", s.host,
		"--api-port", fmt.Sprint(s.port),
		"../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.client, out)
}
