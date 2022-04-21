package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/otel_tracer"
	"github.com/filecoin-project/bacalhau/internal/scheduler/libp2p"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackNode struct {
	ComputeNode   *ComputeNode
	RequesterNode *RequesterNode
	JsonRpcPort   int
	IpfsRepo      string
}

type DevStack struct {
	Nodes []*DevStackNode
}

func NewDevStack(
	ctx context.Context,
	count, badActors int,
) (*DevStack, error) {

	nodes := []*DevStackNode{}

	bacalhauMultiAddresses := []string{}
	devstackIpfsMultiAddresses := []string{}

	// Initialize the root trace for all of Otel
	tp, _ := otel_tracer.GetOtelTP(ctx)
	tracer := tp.Tracer("bacalhau.org")
	_, span := tracer.Start(ctx, "Provisioning Nodes")

	// create 3 bacalhau compute nodes
	for i := 0; i < count; i++ {
		log.Debug().Msgf(`
---------------------
  Creating Node #%d
---------------------`, i)
		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		libp2pScheduler, err := libp2p.NewLibp2pScheduler(ctx, libp2pPort)
		if err != nil {
			return nil, err
		}

		_, requesterNodeSpan := tracer.Start(ctx, fmt.Sprintf("Starting requester Node: %d", i))
		requesterNode, err := NewRequesterNode(ctx, libp2pScheduler)
		if err != nil {
			return nil, err
		}
		requesterNodeSpan.End()

		_, computeNodeSpan := tracer.Start(ctx, fmt.Sprintf("Starting compute Node: %d", i))
		computeNode, err := NewComputeNode(ctx, libp2pScheduler, badActors > i)
		if err != nil {
			return nil, err
		}
		computeNodeSpan.End()

		// at this point the requester and compute nodes are both subscribing to the scheduler events
		_, libp2pSchedulerSpan := tracer.Start(ctx, "Starting Libp2p")
		err = libp2pScheduler.Start()
		if err != nil {
			return nil, err
		}
		libp2pSchedulerSpan.End()

		bacalhauMultiAddress := fmt.Sprintf("%s/p2p/%s", libp2pScheduler.Host.Addrs()[0].String(), libp2pScheduler.Host.ID())
		log.Debug().Msgf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)

		// if we have started any bacalhau servers already, use the first one
		if len(bacalhauMultiAddresses) > 0 {
			err = libp2pScheduler.Connect(bacalhauMultiAddresses[0])
			if err != nil {
				return nil, err
			}
		}

		jsonRpcPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		_, startBacJsonRPCServerSpan := tracer.Start(ctx, "Starting Bac Json Rpc Server")
		RunBacalhauJsonRpcServer(ctx, "0.0.0.0", jsonRpcPort, requesterNode)
		startBacJsonRPCServerSpan.End()

		connectToMultiAddress := ""

		// if we have started any ipfs servers already, use the first one
		if len(devstackIpfsMultiAddresses) > 0 {
			connectToMultiAddress = devstackIpfsMultiAddresses[0]
		}

		_, startBacDevIPFSServerSpan := tracer.Start(ctx, "Starting Bac Dev IPFS Server")
		ipfsRepo, computeNodeIpfsMultiaddresses, err := ipfs.StartBacalhauDevelopmentIpfsServer(ctx, connectToMultiAddress)
		if err != nil {
			log.Error().Err(err).Msg("Unable to start Bacalhau Dev Ipfs Server")
			return nil, err
		}
		startBacDevIPFSServerSpan.End()

		bacalhauMultiAddresses = append(bacalhauMultiAddresses, bacalhauMultiAddress)
		devstackIpfsMultiAddresses = append(devstackIpfsMultiAddresses, computeNodeIpfsMultiaddresses[0])

		computeNode.IpfsRepo = ipfsRepo

		log.Debug().Msgf("bacalhau multiaddress: %s\n", bacalhauMultiAddress)
		log.Debug().Msgf("ipfs multiaddress: %s\n", devstackIpfsMultiAddresses)
		log.Debug().Msgf("ipfs repo: %s\n", ipfsRepo)

		devStackNode := &DevStackNode{
			ComputeNode:   computeNode,
			RequesterNode: requesterNode,
			JsonRpcPort:   jsonRpcPort,
			IpfsRepo:      ipfsRepo,
		}

		nodes = append(nodes, devStackNode)
		log.Debug().Msg("==== Complete")
	}

	stack := &DevStack{
		Nodes: nodes,
	}
	span.End()

	log.Debug().Msg("Finished provisioning nodes.")
	return stack, nil
}

func (stack *DevStack) PrintNodeInfo() {

	debugScriptContent := "#!/bin/bash"
	scriptForDebuggingPath := "/tmp/debug_script.sh"

	logString := `
-------------------------------
environment
-------------------------------	
`
	for nodeNumber, node := range stack.Nodes {

		logString = logString + fmt.Sprintf(`
IPFS_PATH_%d=%s
JSON_PORT_%d=%d`, nodeNumber, node.IpfsRepo, nodeNumber, node.JsonRpcPort)

		debugScriptContent = debugScriptContent + fmt.Sprintf(`
export IPFS_PATH_%d=%s
export JSON_PORT_%d=%d`, nodeNumber, node.IpfsRepo, nodeNumber, node.JsonRpcPort)

	}

	log.Info().Msg(logString + "\n")

	log.Info().Msg(`
-------------------------------
example job
-------------------------------

cid=$( IPFS_PATH=$IPFS_PATH_0 ipfs add -q ./testdata/grep_file.txt )
go run . --jsonrpc-port=$JSON_PORT_0 submit --cids=$cid --commands="grep kiwi /ipfs/$cid"
go run . --jsonrpc-port=$JSON_PORT_0 list

`)
	debugScriptContent = debugScriptContent + `
export cid=$( IPFS_PATH=$IPFS_PATH_0 ipfs add -q ./testdata/grep_file.txt )
`

	if os.Getenv("WRITE_TEMP_SCRIPT") != "" {
		debugScriptFile, err := os.OpenFile(scriptForDebuggingPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0700)
		if err != nil {
			log.Fatal().Msgf("Could not write temporary script for execution")
			return
		}
		_, _ = debugScriptFile.WriteString(debugScriptContent)
	}

}
