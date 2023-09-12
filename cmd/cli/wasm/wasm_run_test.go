//go:build unit || !integration

package wasm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type WasmRunSuite struct {
	cmdtesting.BaseSuite
}

func TestWasmRunSuite(t *testing.T) {
	util.Fatal = util.FakeFatalErrorHandler
	suite.Run(t, new(WasmRunSuite))
}

func (s *WasmRunSuite) Test_SupportsRelativeDirectory() {
	ctx := context.Background()
	_, out, err := cmdtesting.ExecuteTestCobraCommand("wasm", "run",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutputLegacy(ctx, s.T(), s.Client, out)
}
