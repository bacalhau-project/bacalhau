package config

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config"
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
		Short: "List all config keys, values, and their descriptions",
		Long: `The config list command displays all available configuration keys along with their detailed descriptions.
This comprehensive list helps you understand the settings you can adjust to customize the bacalhau's behavior. 
Each key shown can be used with: 
- bacalhau config set <key> <value> to directly set the value
- bacalhau --config=<key>=<value> to temporarily modify the setting for a single command execution`,
		Args:     cobra.MinimumNArgs(0),
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return err
			}
			log.Debug().Msgf("Config loaded from: %s, and with data-dir %s", cfg.Paths(), cfg.Get(types.DataDirKey))
			return list(cmd, cfg, o)
		},
	}
	listCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o))
	return listCmd
}

type configListEntry struct {
	Key         string
	Value       any
	Description string
}

func list(cmd *cobra.Command, cfg *config.Config, o output.OutputOptions) error {
	if o.Format == output.YAMLFormat {
		var out types.Bacalhau
		if err := cfg.Unmarshal(&out); err != nil {
			return err
		}
		bytes, err := yaml.Marshal(out)
		if err != nil {
			return err
		}
		stdout := cmd.OutOrStdout()
		_, _ = fmt.Fprintln(stdout, string(bytes))
		return nil
	}
	o.SortBy = []table.SortBy{{
		Name: "Key",
		Mode: table.Asc,
	}}
	var cfgList []configListEntry
	for key, description := range types.ConfigDescriptions {
		cfgList = append(cfgList, configListEntry{
			Key:         key,
			Value:       cfg.Get(key),
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
		ColumnConfig: table.ColumnConfig{Name: "Value", WidthMax: 80, WidthMaxEnforcer: text.WrapSoft},
		Value: func(s configListEntry) string {
			return fmt.Sprintf("%v", s.Value)
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Description", WidthMax: 80, WidthMaxEnforcer: text.WrapSoft},
		Value: func(v configListEntry) string {
			return fmt.Sprintf("%v", v.Description)
		},
	},
}
