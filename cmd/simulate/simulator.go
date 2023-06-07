package simulate

import (
	"fmt"

	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/handler"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "simulator",
		Short: "Run the bacalhau simulator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err, exitcode := runSimulator(cmd); err != nil {
				handler.Fatal(cmd, err, exitcode)
			}
			return nil
		},
	}
}

func runSimulator(cmd *cobra.Command) (error, int) {
	ctx := cmd.Context()
	cm := handler.GetCleanupManager(ctx)
	//Cleanup manager ensures that resources are freed before exiting:
	datastore := inmemory.NewJobStore()
	libp2pHost, err := libp2p.NewHost(9075) //nolint:gomnd
	if err != nil {
		return fmt.Errorf("error creating libp2p host: %w", err), handler.ExitError
	}
	cm.RegisterCallback(libp2pHost.Close)

	// print out simulator multi-address
	p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + libp2pHost.ID().String())
	if err != nil {
		return fmt.Errorf("error creating p2p multiaddr: %w", err), handler.ExitError
	}
	fullAddr := libp2pHost.Addrs()[0].Encapsulate(p2pAddr)
	log.Ctx(ctx).Info().Msgf("Simulator reachable at: %s", fullAddr)
	log.Ctx(ctx).Info().Msgf("You can run: bacalhau devstack --simulator-addr \"%s\"", fullAddr)

	// Create node config from cmd arguments
	nodeConfig := node.NodeConfig{
		CleanupManager: cm,
		JobStore:       datastore,
		Host:           libp2pHost,
		HostAddress:    "0.0.0.0",
		// TODO(forrest)
		//APIPort:             cmd2.apiPort,
		ComputeConfig:       node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig: node.NewRequesterConfigWithDefaults(),
		SimulatorNodeID:     libp2pHost.ID().String(),
		IsComputeNode:       true,
		IsRequesterNode:     true,
		DependencyInjector:  devstack.NewNoopNodeDependencyInjector(),
	}
	node, err := node.NewNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("error creating node: %w", err), handler.ExitError
	}
	// Start node
	err = node.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting node: %w", err), handler.ExitError
	}

	<-ctx.Done() // block until killed
	return nil, handler.ExitSuccess
}
