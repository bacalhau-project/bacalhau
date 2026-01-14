package profile

import (
	"fmt"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// profileListEntry represents a profile entry for display
type profileListEntry struct {
	Current  string `json:"current"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Auth     string `json:"auth"`
}

// ListOptions contains options for the list command
type ListOptions struct {
	output.OutputOptions
}

// NewListOptions returns initialized ListOptions
func NewListOptions() *ListOptions {
	return &ListOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
	}
}

func NewListCmd() *cobra.Command {
	o := NewListOptions()
	listCmd := &cobra.Command{
		Use:           "list",
		Short:         "List all CLI profiles.",
		Long:          `List all configured CLI connection profiles for Bacalhau clusters.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       hook.ClientPreRunHooks,
		PostRunE:      hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}
			return o.run(cmd, cfg)
		},
	}

	listCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return listCmd
}

var listColumns = []output.TableColumn[profileListEntry]{
	{
		ColumnConfig: table.ColumnConfig{Name: "CURRENT"},
		Value:        func(p profileListEntry) string { return p.Current },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "NAME"},
		Value:        func(p profileListEntry) string { return p.Name },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "ENDPOINT"},
		Value:        func(p profileListEntry) string { return p.Endpoint },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "AUTH"},
		Value:        func(p profileListEntry) string { return p.Auth },
	},
}

func (o *ListOptions) run(cmd *cobra.Command, cfg *config.Config) error {
	dataDir, ok := cfg.Get(types.DataDirKey).(string)
	if !ok {
		return fmt.Errorf("data directory configuration is invalid")
	}
	profilesDir := filepath.Join(dataDir, "profiles")
	store := profile.NewStore(profilesDir)

	names, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(names) == 0 {
		cmd.Println("No profiles found")
		return nil
	}

	current, err := store.GetCurrent()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	var entries []profileListEntry
	for _, name := range names {
		p, err := store.Load(name)
		if err != nil {
			return fmt.Errorf("failed to load profile %q: %w", name, err)
		}

		currentMarker := ""
		if name == current {
			currentMarker = "*"
		}

		authStatus := ""
		if p.Auth != nil && p.Auth.Token != "" {
			authStatus = "token"
		}

		entries = append(entries, profileListEntry{
			Current:  currentMarker,
			Name:     name,
			Endpoint: p.Endpoint,
			Auth:     authStatus,
		})
	}

	if err = output.Output(cmd, listColumns, o.OutputOptions, entries); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
