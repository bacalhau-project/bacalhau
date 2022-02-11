package bacalhau

import (
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
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
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()

		computeNode, err := internal.NewComputeNode(ctx, hostPort)
		if err != nil {
			return err
		}
		err = computeNode.Connect(peerConnect)
		if err != nil {
			return err
		}
		jsonRpcString := ""
		devString := ""
		ipfsGatewayPort, err := freeport.GetFreePort()
		if err != nil {
			return err
		}
		ipfsApiPort, err := freeport.GetFreePort()
		if err != nil {
			return err
		}

		if developmentMode {
			jsonRpcPort, err := freeport.GetFreePort()
			if err != nil {
				return err
			}
			jsonRpcString = fmt.Sprintf(" --jsonrpc-port %d", jsonRpcPort)

			devString = " --dev"
		}
		if startIpfsDevOnly {
			if computeNode.TempIpfsRepo {
				ipfs.Init(computeNode.IpfsRepo)
			}
			ipfs.StartDaemon(computeNode.IpfsRepo, ipfsGatewayPort, ipfsApiPort)
			devString += " --start-ipfs-dev-only"
		}
		type TemplateContents struct {
			HostAddress     string
			HostPort        int
			ComputeNodeId   string
			JsonRpcString   string
			DevString       string
			IpfsPath        string
			IpfsGatewayPort int
			IpfsApiPort     int
		}
		td := TemplateContents{
			HostAddress:     hostAddress,
			HostPort:        hostPort,
			ComputeNodeId:   computeNode.Id,
			JsonRpcString:   jsonRpcString,
			DevString:       devString,
			IpfsPath:        computeNode.IpfsRepo,
			IpfsGatewayPort: ipfsGatewayPort,
			IpfsApiPort:     ipfsApiPort,
		}

		t, err := template.New("msg").Parse(
			`Command to connect other peers:

go run . serve --peer /ip4/{{.HostAddress}}/tcp/{{.HostPort}}/p2p/{{.ComputeNodeId}}{{.JsonRpcString}}{{.DevString}}

To pin some files locally in the ipfs daemon you started (if you used --start-ipfs-dev-only):

cid=$(IPFS_PATH={{.IpfsPath}} ipfs add -q /etc/passwd)

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

		internal.RunBacalhauRpcServer(hostAddress, jsonrpcPort, computeNode)

		return nil

	},
}
