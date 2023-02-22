package bacalhau

import (
	"encoding/json"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/libp2p"
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

func id(_ *cobra.Command, OS *ServeOptions) error {
	libp2pHost, err := libp2p.NewHost(OS.SwarmPort)
	if err != nil {
		return err
	}

	info := IDInfo{
		ID: libp2pHost.ID().String(),
	}
	_ = libp2pHost.Close()

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if err := enc.Encode(info); err != nil {
		return err
	}

	return nil
}
