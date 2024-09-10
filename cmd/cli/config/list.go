package config

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func newListCmd() *cobra.Command {
	o := output.OutputOptions{
		Format:     output.TableFormat,
		Pretty:     true,
		HideHeader: false,
		NoStyle:    false,
		Wide:       false,
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all config keys and their descriptions",
		Long: `The config list command displays all available configuration keys along with their detailed descriptions.
This comprehensive list helps you understand the settings you can adjust to customize the bacalhau's behavior. 
Each key shown can be used with: 
- bacalhau config set <key> <value> to directly set the value
- bacalhau --config=<key>=<value> to temporarily modify the setting for a single command execution`,
		Args:     cobra.MinimumNArgs(0),
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cmd, o)
		},
	}
	listCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o))
	return listCmd
}

type configListEntry struct {
	Key         string
	Description string
}

func list(cmd *cobra.Command, o output.OutputOptions) error {
	o.SortBy = []table.SortBy{{
		Name: "Key",
		Mode: table.Asc,
	}}
	var cfgList []configListEntry
	for key, description := range types.ConfigDescriptions {
		cfgList = append(cfgList, configListEntry{
			Key:         key,
			Description: description,
		})
	}

	if err := output.Output(cmd, listColumns, o, cfgList); err != nil {
		return err
	}

	return nil
}

var listColumns = []output.TableColumn[configListEntry]{
	{
		ColumnConfig: table.ColumnConfig{Name: "Key"},
		Value: func(s configListEntry) string {
			return s.Key
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Description", WidthMax: 80, WidthMaxEnforcer: text.WrapText},
		Value: func(v configListEntry) string {
			return fmt.Sprintf("%v", v.Description)
		},
	},
}
