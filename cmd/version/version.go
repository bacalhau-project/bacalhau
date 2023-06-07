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
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/handler"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

// Versions is a struct for version information
type Versions struct {
	ClientVersion *model.BuildVersionInfo `json:"clientVersion,omitempty"`
	ServerVersion *model.BuildVersionInfo `json:"serverVersion,omitempty"`
}

// VersionOptions is a struct to support version command
type VersionOptions struct {
	ClientOnly bool
	Output     string

	args []string
}

// NewVersionOptions returns initialized Options
func NewVersionOptions() *VersionOptions {
	return &VersionOptions{}
}

func NewCmd() *cobra.Command {
	oV := NewVersionOptions()

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Get the client and server version.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err, exitcode := runVersion(cmd, oV); err != nil {
				handler.Fatal(cmd, err, exitcode)
			}
			return nil
		},
	}
	versionCmd.Flags().BoolVar(&oV.ClientOnly, "client", oV.ClientOnly, "If true, shows client version only (no server required).")
	versionCmd.Flags().StringVarP(&oV.Output, "output", "o", oV.Output, "One of 'yaml' or 'json'.")

	return versionCmd
}

func runVersion(cmd *cobra.Command, oV *VersionOptions) (error, int) {
	ctx := cmd.Context()

	oV.Output = strings.TrimSpace(strings.ToLower(oV.Output))

	err := oV.Validate(cmd)
	if err != nil {
		return fmt.Errorf("error validating version: %w", err), handler.ExitError
	}

	err = oV.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("error running version: %w", err), handler.ExitError
	}

	return nil, handler.ExitSuccess
}

// Validate validates the provided options
func (oV *VersionOptions) Validate(*cobra.Command) error {
	if len(oV.args) != 0 {
		return fmt.Errorf("extra arguments: %v", oV.args)
	}

	// TODO constants for json and yaml
	if oV.Output != "" && oV.Output != "yaml" && oV.Output != "json" {
		return errors.New(`--output must be 'yaml' or 'json'`)
	}

	return nil
}

// Run executes version command
func (oV *VersionOptions) Run(ctx context.Context, cmd *cobra.Command) error {
	var (
		versions Versions
	)

	versions.ClientVersion = version.Get()

	if !oV.ClientOnly {
		serverVersion, err := handler.GetAPIClient(ctx).Version(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("could not get server version")
			return err
		}

		versions.ServerVersion = serverVersion
	}

	switch oV.Output {
	case "":
		cmd.Printf("Client Version: %s\n", versions.ClientVersion.GitVersion)
		if versions.ServerVersion != nil {
			cmd.Printf("Server Version: %s\n", versions.ServerVersion.GitVersion)
		}
		// TODO(forrest) constans for cases
	case "yaml":
		marshaled, err := model.YAMLMarshalWithMax(versions)
		if err != nil {
			return err
		}
		cmd.Println(string(marshaled))
	case "json":
		marshaled, err := model.JSONMarshalWithMax(versions)
		if err != nil {
			return err
		}
		cmd.Println(string(marshaled))
	default:
		// There is a bug in the program if we hit this case.
		// However, we follow a policy of never panicking.
		return fmt.Errorf("VersionOptions were not validated: --output=%q should have been rejected", oV.Output)
	}

	return nil
}
