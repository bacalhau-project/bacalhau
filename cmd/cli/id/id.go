package id

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

type IDInfo struct {
	ID       string `json:"ID"`
	ClientID string `json:"ClientID"`
}

func NewCmd() *cobra.Command {
	outputOpts := output.OutputOptions{
		Format: output.JSONFormat,
	}

	idFlags := map[string][]configflags.Definition{}

	idCmd := &cobra.Command{
		Use:   "id",
		Short: "Show bacalhau node id info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			return id(cmd, cfg, outputOpts)
		},
	}

	idCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&outputOpts))

	if err := configflags.RegisterFlags(idCmd, idFlags); err != nil {
		util.Fatal(idCmd, err, 1)
	}

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

func id(cmd *cobra.Command, cfg types.BacalhauConfig, outputOpts output.OutputOptions) error {
	clientID, err := config.GetClientID(cfg.User.KeyPath)
	if err != nil {
		return err
	}
	info := IDInfo{
		ID:       cfg.Node.Name,
		ClientID: clientID,
	}

	return output.OutputOne(cmd, idColumns, outputOpts, info)
}
