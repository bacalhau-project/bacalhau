package bacalhau

import (
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/scheduler/libp2p"
	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

var peerConnect string
var hostAddress string
var hostPort int
var startIpfsDevOnly bool

func init() {
	serveCmd.PersistentFlags().StringVar(
		&peerConnect, "peer", "",
		`The libp2p multiaddress to connect to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&hostAddress, "host", "0.0.0.0",
		`The port to listen on.`,
	)
	serveCmd.PersistentFlags().IntVar(
		&hostPort, "port", 0,
		`The port to listen on.`,
	)
	serveCmd.PersistentFlags().BoolVar(
		&startIpfsDevOnly, "start-ipfs-dev-only", false,
		`Start an ipfs node in a bacalhau-node specific data directory,`+
			` FOR DEV PURPOSES ONLY (in production, run a single bacalhau server`+
			` on servers where you already have ipfs servers running and it will`+
			` use their default data directories).`,
	)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the bacalhau compute node",
	RunE: func(cmd *cobra.Command, args []string) error { // nolint

		ctx := context.Background()

		libp2pScheduler, err := libp2p.NewLibp2pScheduler(ctx, hostPort)
		if err != nil {
			return err
		}

		requesterNode, err := internal.NewRequesterNode(ctx, libp2pScheduler)
		if err != nil {
			return err
		}
		computeNode, err := internal.NewComputeNode(ctx, libp2pScheduler, false)
		if err != nil {
			return err
		}
		err = libp2pScheduler.Connect(peerConnect)
		if err != nil {
			return err
		}
		jsonRpcString := ""
		devString := ""
		if developmentMode {
			jsonRpcPort, err := freeport.GetFreePort()
			if err != nil {
				return err
			}
			jsonRpcString = fmt.Sprintf(" --jsonrpc-port %d", jsonRpcPort)
			devString = " --dev"
		}

		ipfsPathDevString := " "
		if startIpfsDevOnly {
			ipfsRepo, _, err := ipfs.StartBacalhauDevelopmentIpfsServer(ctx, "")
			if err != nil {
				return err
			}
			computeNode.IpfsRepo = ipfsRepo
			ipfsPathDevString = fmt.Sprintf("IPFS_PATH=%s ", ipfsRepo)
			devString += " --start-ipfs-dev-only"
		}
		type TemplateContents struct {
			HostAddress       string
			HostPort          int
			ComputeNodeId     string
			JsonRpcString     string
			DevString         string
			IpfsPathDevString string
		}

		hostId, err := computeNode.Scheduler.HostId()
		if err != nil {
			return err
		}

		td := TemplateContents{
			HostAddress:       hostAddress,
			HostPort:          hostPort,
			ComputeNodeId:     hostId,
			JsonRpcString:     jsonRpcString,
			DevString:         devString,
			IpfsPathDevString: ipfsPathDevString,
		}

		if developmentMode {
			td.HostPort = 8080
			td.HostAddress = "127.0.0.1"
		}

		t, err := template.New("msg").Parse(
			`Command to connect other peers:

go run . serve --peer /ip4/{{.HostAddress}}/tcp/{{.HostPort}}/p2p/{{.ComputeNodeId}}{{.JsonRpcString}}{{.DevString}}

To pin some files locally in the ipfs daemon you started (if you used --start-ipfs-dev-only):

cid=$({{.IpfsPathDevString}}ipfs add -q /etc/passwd)

To submit a job that uses that data (and so should be preferentially scheduled on this node):

go run . submit --cids=$cid --commands="grep admin /ipfs/$cid"

`,
		)
		if err != nil {
			return err
		}
		err = t.Execute(os.Stdout, td)
		if err != nil {
			return err
		}

		internal.RunBacalhauJsonRpcServer(ctx, hostAddress, jsonrpcPort, requesterNode)

		// wait forever because everything else is running in a goroutine
		select {}
	},
}
