package bacalhau

import (
	"encoding/json"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/system"
	libp2p_transport "github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/multiformats/go-multiaddr"

	"github.com/spf13/cobra"
)

type IDInfo struct {
	ID string `json:"ID"`
}

func newIDCmd() *cobra.Command {
	OS := NewServeOptions()

	idCmd := &cobra.Command{
		Use:   "id",
		Short: "Show bacalhau node id info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return id(cmd, OS)
		},
	}

	setupLibp2pCLIFlags(idCmd, OS)

	return idCmd
}

func id(cmd *cobra.Command, OS *ServeOptions) error {
	// Cleanup manager ensures that resources are freed before exiting:
	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTraceProvider)
	defer cm.Cleanup()
	ctx := cmd.Context()

	libp2pHost, err := libp2p.NewHost(ctx, cm, OS.SwarmPort, []multiaddr.Multiaddr{})
	if err != nil {
		return err
	}
	transport, err := libp2p_transport.NewTransport(ctx, cm, libp2pHost)
	if err != nil {
		return err
	}

	info := IDInfo{
		ID: transport.HostID(),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if err := enc.Encode(info); err != nil {
		return err
	}

	return nil
}
