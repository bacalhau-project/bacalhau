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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/version"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

// Versions is a struct for version information
type Versions struct {
	ClientVersion *models.BuildVersionInfo `json:"clientVersion,omitempty"`
	ServerVersion *models.BuildVersionInfo `json:"serverVersion,omitempty"`
}

// VersionOptions is a struct to support version command
type VersionOptions struct {
	ClientOnly bool
	OutputOpts output.OutputOptions
}

// NewVersionOptions returns initialized Options
func NewVersionOptions() *VersionOptions {
	return &VersionOptions{
		OutputOpts: output.OutputOptions{Format: output.TableFormat},
	}
}

func NewCmd() *cobra.Command {
	oV := NewVersionOptions()

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Get the client and server version.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			if err := runVersion(cmd, oV); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}
	versionCmd.Flags().BoolVar(&oV.ClientOnly, "client", oV.ClientOnly, "If true, shows client version only (no server required).")
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

var clientVersionColumn = output.TableColumn[Versions]{
	ColumnConfig: table.ColumnConfig{Name: "client"},
	Value:        func(v Versions) string { return v.ClientVersion.GitVersion },
}

var serverVersionColumn = output.TableColumn[Versions]{
	ColumnConfig: table.ColumnConfig{Name: "server"},
	Value:        func(v Versions) string { return v.ServerVersion.GitVersion },
}

// Run executes version command
func (oV *VersionOptions) Run(ctx context.Context, cmd *cobra.Command) error {
	var (
		versions Versions
		columns  []output.TableColumn[Versions]
	)

	versions.ClientVersion = version.Get()
	columns = append(columns, clientVersionColumn)

	if !oV.ClientOnly {
		serverVersion, err := util.GetAPIClient(ctx).Version(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("could not get server version")
			// Fail silently
		} else {
			versions.ServerVersion = serverVersion
			columns = append(columns, serverVersionColumn)
		}
	}

	// Assuming versions.ServerVersion may be nil if oV.ClientOnly is true
	var serverVersion string
	if versions.ServerVersion != nil {
		serverVersion = versions.ServerVersion.GitVersion
	}

	checkForUpdates(ctx, versions.ClientVersion.GitVersion, serverVersion)

	return output.OutputOne(cmd, columns, oV.OutputOpts, versions)
}

type ServerResponse struct {
	Version string `json:"version"`
	Message string `json:"message"`
}

func checkForUpdates(ctx context.Context, currentClientVersion, currentServerVersion string) {
	// Construct URL with query parameters
	u, err := url.Parse("http://update.bacalhau.org/version")
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to parse URL.")
		return
	}

	q := u.Query()
	q.Set("clientVersion", currentClientVersion)
	if currentServerVersion != "" {
		q.Set("serverVersion", currentServerVersion)
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to fetch the latest version from the server.")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to read response body.")
		return
	}

	var serverResponse ServerResponse
	err = json.Unmarshal(body, &serverResponse)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the server response.")
		return
	}

	if currentClientVersion != serverResponse.Version {
		fmt.Println(serverResponse.Message)
	} else {
		fmt.Println("Your Bacalhau Version is latest")
	}
}