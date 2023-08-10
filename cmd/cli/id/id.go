package id

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/libp2p/go-libp2p"

	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type IDInfo struct {
	ID       string `json:"ID"`
	ClientID string `json:"ClientID"`
}

func NewCmd() *cobra.Command {
	OS := serve.NewServeOptions()
	outputOpts := output.OutputOptions{
		Format: output.JSONFormat,
	}

	// make sure serve options point to local mode
	OS.PeerConnect = serve.DefaultPeerConnect
	OS.PrivateInternalIPFS = true

	idCmd := &cobra.Command{
		Use:   "id",
		Short: "Show bacalhau node id info",
		Run: func(cmd *cobra.Command, _ []string) {
			if err := id(cmd, OS, outputOpts); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	idCmd.Flags().AddFlagSet(flags.OutputFormatFlags(&outputOpts))
	serve.SetupLibp2pCLIFlags(idCmd, OS)

	return idCmd
}

var idColumns = []output.TableColumn[IDInfo]{
	{
		ColumnConfig: table.ColumnConfig{Name: "id"},
		Value:        func(i IDInfo) string { return i.ID },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "client id"},
		Value:        func(i IDInfo) string { return i.ClientID },
	},
}

func id(cmd *cobra.Command, OS *serve.ServeOptions, outputOpts output.OutputOptions) error {
	fsRepo, err := repo.NewFS(viper.GetString("repo"))
	if err != nil {
		return err
	}
	privKey, err := fsRepo.InitLibp2pPrivateKey(OS.SwarmPort)
	if err != nil {
		return err
	}
	libp2pHost, err := bac_libp2p.NewHost(OS.SwarmPort, libp2p.Identity(privKey))
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError("libp2pHost", libp2pHost)

	info := IDInfo{
		ID:       libp2pHost.ID().String(),
		ClientID: system.GetClientID(),
	}

	return output.OutputOne(cmd, idColumns, outputOpts, info)
}
