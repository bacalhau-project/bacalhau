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
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/version"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"

	"github.com/bacalhau-project/bacalhau/pkg/config"
)

type Versions struct {
	ClientVersion *models.BuildVersionInfo `json:"clientVersion,omitempty"`
	ServerVersion *models.BuildVersionInfo `json:"serverVersion,omitempty"`
	LatestVersion *models.BuildVersionInfo `json:"latestVersion,omitempty"`
	UpdateMessage string                   `json:"updateMessage,omitempty"`
}

type VersionOptions struct {
	ClientOnly bool
	OutputOpts output.OutputOptions
}

func NewVersionOptions() *VersionOptions {
	return &VersionOptions{
		OutputOpts: output.OutputOptions{Format: output.TableFormat},
	}
}

func NewCmd() *cobra.Command {
	oV := NewVersionOptions()

	versionCmd := &cobra.Command{
		Use:    "version",
		Short:  "Get the client and server version.",
		Args:   cobra.NoArgs,
		PreRun: util.ApplyPorcelainLogLevel,
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

var latestVersionColumn = output.TableColumn[Versions]{
	ColumnConfig: table.ColumnConfig{Name: "latest"},
	Value:        func(v Versions) string { return v.LatestVersion.GitVersion },
}

var updateMessageColumn = output.TableColumn[Versions]{
	ColumnConfig: table.ColumnConfig{Name: "Update Message"},
	Value:        func(v Versions) string { return v.UpdateMessage },
}

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
			return fmt.Errorf("error running version: %w", err)
		}

		versions.ServerVersion = serverVersion
		columns = append(columns, serverVersionColumn)

		clientID, err := config.GetClientID()
		if err != nil {
			return fmt.Errorf("error getting UserID: %w", err)
		}

		updateCheck, err := checkForUpdates(ctx, versions.ClientVersion, versions.ServerVersion, clientID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("Failed to perform update check")
		} else {
			versions.UpdateMessage = updateCheck.Message
			versions.LatestVersion = updateCheck.Version
			columns = append(columns, latestVersionColumn)

			// Print the update message only if --output flag is not used
			if oV.OutputOpts.Format == output.TableFormat {
				fmt.Println(updateCheck.Message)
			} else {
				columns = append(columns, updateMessageColumn)
			}
		}
	}

	return output.OutputOne(cmd, columns, oV.OutputOpts, versions)
}

type serverResponse struct {
	Version *models.BuildVersionInfo `json:"version"`
	Message string                   `json:"message"`
}

func checkForUpdates(
	ctx context.Context,
	currentClientVersion, currentServerVersion *models.BuildVersionInfo,
	clientID string,
) (*serverResponse, error) {
	u, err := url.Parse("http://update.bacalhau.org/version")
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse URL")
	}

	q := u.Query()
	q.Set("clientVersion", currentClientVersion.GitVersion)
	if currentServerVersion.GitVersion != "" {
		q.Set("serverVersion", currentServerVersion.GitVersion)
	}
	q.Set("operatingSystem", currentClientVersion.GOOS)
	q.Set("architecture", currentClientVersion.GOARCH)
	q.Set("userID", clientID)

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build HTTP request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the latest version from the server")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var updateCheck serverResponse
	err = json.Unmarshal(body, &updateCheck)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the server response when checking for updates")
	}

	return &updateCheck, nil
}
