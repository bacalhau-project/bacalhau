package id

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type IDInfo struct {
	ID       string `json:"ID"`
	ClientID string `json:"ClientID"`
}

func NewCmd() *cobra.Command {
	OS := serve.NewServeOptions()

	// make sure serve options point to local mode
	OS.PeerConnect = serve.DefaultPeerConnect
	OS.PrivateInternalIPFS = true

	idCmd := &cobra.Command{
		Use:   "id",
		Short: "Show bacalhau node id info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return id(cmd, OS)
		},
	}

	serve.SetupLibp2pCLIFlags(idCmd, OS)

	return idCmd
}

func id(_ *cobra.Command, OS *serve.ServeOptions) error {
	libp2pHost, err := libp2p.NewHost(OS.SwarmPort)
	if err != nil {
		return err
	}

	info := IDInfo{
		ID:       libp2pHost.ID().String(),
		ClientID: system.GetClientID(),
	}
	_ = libp2pHost.Close()

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if err := enc.Encode(info); err != nil {
		return err
	}

	return nil
}
