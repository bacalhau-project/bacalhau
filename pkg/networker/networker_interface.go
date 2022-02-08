package networker

import (
	"os"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/spf13/cobra"
)

type NetworkerInterface interface {
	// Run a bacalhau server
	RunBacalhauRpcServer(string, int, *internal.ComputeNode) error

	// Instantiation getter/setters
	GetCmd() *cobra.Command
	SetCmd(*cobra.Command)
	GetCmdArgs() []string
	SetCmdArgs([]string)
}

func GetNetworker(cmd *cobra.Command, args []string) NetworkerInterface {
	var n NetworkerInterface = &NetworkerLive{}
	if os.Getenv("TEST_PASS") != "" {
		n = &NetworkerMock{}
	}
	n.SetCmd(cmd)
	n.SetCmdArgs(args)
	return n
}