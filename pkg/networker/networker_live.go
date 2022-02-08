package networker

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/spf13/cobra"
)

type NetworkerLive struct {
	_cmd            *cobra.Command
	_cmdArgs        []string
}


func (i *NetworkerLive) RunBacalhauRpcServer(host string, port int, computeNode *internal.ComputeNode) error {
	job := &internal.JobServer{
		ComputeNode: computeNode,
	}
	err := rpc.Register(job)
	if err != nil {
		log.Fatalf("Format of service Job isn't correct. %s", err)
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatalf("Couldn't start listening on port %d. Error %s", port, err)
		return err
	}
	log.Println("Serving RPC handler")
	err = http.Serve(l, nil)
	if err != nil {
		log.Fatalf("Error serving: %s", err)
		return err
	}
	return nil
}


func (n *NetworkerLive) GetCmd() *cobra.Command {
	return n._cmd
}

func (n *NetworkerLive) SetCmd(cmd *cobra.Command) {
	n._cmd = cmd
}

func (n *NetworkerLive) GetCmdArgs() []string {
	return n._cmdArgs
}

func (n *NetworkerLive) SetCmdArgs(args []string) {
	n._cmdArgs = args
}

