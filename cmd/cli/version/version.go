/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package version

import (
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/version"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

type VersionOptions struct {
	Client     bool
	Server     bool
	OutputOpts output.OutputOptions
}

func NewVersionOptions() *VersionOptions {
	return &VersionOptions{
		Client:     true,
		Server:     false,
		OutputOpts: output.OutputOptions{Format: output.TableFormat},
	}
}

func NewCmd() *cobra.Command {
	oV := NewVersionOptions()

	versionCmd := &cobra.Command{
		Use:    "version",
		Short:  "Get the client and optionally the server (requester) version if specified",
		Args:   cobra.NoArgs,
		PreRun: hook.ApplyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion(cmd, oV)
		},
	}
	fset := pflag.NewFlagSet("version", pflag.ContinueOnError)

	// NB(forrest): the client flag remains present for backwards compatibility with the install
	// script used by https://github.com/bacalhau-project/get.bacalhau.org
	fset.BoolVar(&oV.Client, "client", oV.Client, "If true, shows client version only (no server required).")
	// we are marking the client flag as deprecated, it will still function but will be hidden
	if err := fset.MarkDeprecated("client", "use --server"); err != nil {
		panic(fmt.Sprintf("DEVELOPER ERROR: %s", err))
	}

	fset.BoolVar(&oV.Server, "server", oV.Server, "If true, queries the server (requester) for its version.")
	versionCmd.Flags().AddFlagSet(fset)
	versionCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&oV.OutputOpts))

	return versionCmd
}

func runVersion(cmd *cobra.Command, oV *VersionOptions) error {
	ctx := cmd.Context()

	err := oV.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("error running version: %w", err)
	}

	return err
}

var clientVersionColumn = output.TableColumn[util.Versions]{
	ColumnConfig: table.ColumnConfig{Name: "client"},
	Value:        func(v util.Versions) string { return v.ClientVersion.GitVersion },
}

var serverVersionColumn = output.TableColumn[util.Versions]{
	ColumnConfig: table.ColumnConfig{Name: "server"},
	Value:        func(v util.Versions) string { return v.ServerVersion.GitVersion },
}

var latestVersionColumn = output.TableColumn[util.Versions]{
	ColumnConfig: table.ColumnConfig{Name: "latest"},
	Value:        func(v util.Versions) string { return v.LatestVersion.GitVersion },
}

var updateMessageColumn = output.TableColumn[util.Versions]{
	ColumnConfig: table.ColumnConfig{Name: "Update Message"},
	Value:        func(v util.Versions) string { return v.UpdateMessage },
}

func (oV *VersionOptions) Run(ctx context.Context, cmd *cobra.Command) error {
	var (
		versions util.Versions
		columns  []output.TableColumn[util.Versions]
	)

	// NB(forrest): the client flag remains present for backwards compatibility with the install
	// script used by https://github.com/bacalhau-project/get.bacalhau.org
	if oV.Client == false {
		oV.Server = true
	}
	if oV.Server {
		var err error
		versions, err = util.GetAllVersions(ctx)
		if err != nil {
			// No error on fail of version check. Just print as much as we can.
			cmd.PrintErrln("failed to get server version: ", err)
		}
	}
	versions.ClientVersion = version.Get()

	if versions.ClientVersion != nil {
		columns = append(columns, clientVersionColumn)
	}
	if versions.ServerVersion != nil {
		columns = append(columns, serverVersionColumn)
	}
	if versions.LatestVersion != nil {
		columns = append(columns, latestVersionColumn)
	}

	// Print the update message only if --output flag is not used
	if oV.OutputOpts.Format == output.TableFormat && versions.UpdateMessage != "" {
		cmd.Println(versions.UpdateMessage)
	} else {
		columns = append(columns, updateMessageColumn)
	}

	return output.OutputOne(cmd, columns, oV.OutputOpts, versions)
}
