package networker

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/pkg/mocks"
	"github.com/filecoin-project/bacalhau/pkg/utils"
	"github.com/spf13/cobra"
)

type NetworkerMock struct {
	_cmd            *cobra.Command
	_cmdArgs        []string
}

func (i *NetworkerMock) RunBacalhauRpcServer(host string, port int, computeNode *internal.ComputeNode) error {
	if utils.ContainsString(i.GetCmdArgs(), mocks.CALL_RUN_BACALHAU_RPC_SERVER_SUCCESSFUL_PROBE) {
		return nil
	}

	return fmt.Errorf("Was not able to successfully call pipeline")
}


func (n *NetworkerMock) GetCmd() *cobra.Command {
	return n._cmd
}

func (n *NetworkerMock) SetCmd(cmd *cobra.Command) {
	n._cmd = cmd
}

func (n *NetworkerMock) GetCmdArgs() []string {
	return n._cmdArgs
}

func (n *NetworkerMock) SetCmdArgs(args []string) {
	n._cmdArgs = args
}