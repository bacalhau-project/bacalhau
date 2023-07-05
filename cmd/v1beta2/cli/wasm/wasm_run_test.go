//go:build unit || !integration

package wasm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting2 "github.com/bacalhau-project/bacalhau/cmd/v1beta2/testing"
	util2 "github.com/bacalhau-project/bacalhau/cmd/v1beta2/util"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type WasmRunSuite struct {
	cmdtesting2.BaseSuite
}

func TestWasmRunSuite(t *testing.T) {
	util2.Fatal = util2.FakeFatalErrorHandler
	suite.Run(t, new(WasmRunSuite))
}

func (s *WasmRunSuite) Test_SupportsRelativeDirectory() {
	ctx := context.Background()
	_, out, err := cmdtesting2.ExecuteTestCobraCommand("wasm", "run",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.Client, out)
}
