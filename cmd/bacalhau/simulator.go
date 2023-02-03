package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newSimulatorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "simulator",
		Short: "Run the bacalhau simulator",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSimulator(cmd)
		},
	}
}

func runSimulator(cmd *cobra.Command) error {
	//Cleanup manager ensures that resources are freed before exiting:
	cm := system.NewCleanupManager()
	//cm.RegisterCallback(telemetry.Cleanup)
	defer cm.Cleanup()
	ctx := cmd.Context()
	datastore, err := inmemory.NewInMemoryDatastore()
	if err != nil {
		return err
	}
	libp2pHost, err := libp2p.NewHost(9075) //nolint:gomnd
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error creating libp2p host: %s", err), 1)
	}
	cm.RegisterCallback(func() error {
		return libp2pHost.Close()
	})

	// print out simulator multi-address
	p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + libp2pHost.ID().String())
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error creating p2p multiaddr: %s", err), 1)
	}
	fullAddr := libp2pHost.Addrs()[0].Encapsulate(p2pAddr)
	log.Info().Msgf("Simulator reachable at: %s", fullAddr)
	log.Info().Msgf("You can run: bacalhau devstack --simulator-addr \"%s\"", fullAddr)

	// Create node config from cmd arguments
	nodeConfig := node.NodeConfig{
		CleanupManager:      cm,
		LocalDB:             datastore,
		Host:                libp2pHost,
		HostAddress:         "0.0.0.0",
		APIPort:             apiPort,
		ComputeConfig:       node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig: node.NewRequesterConfigWithDefaults(),
		SimulatorNodeID:     libp2pHost.ID().String(),
		IsComputeNode:       true,
		IsRequesterNode:     true,
	}
	node, err := node.NewNode(ctx, nodeConfig, devstack.NewNoopNodeDependencyInjector())
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error creating node: %s", err), 1)
	}
	// Start node
	err = node.Start(ctx)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error starting node: %s", err), 1)
	}

	<-ctx.Done() // block until killed
	return nil
}
