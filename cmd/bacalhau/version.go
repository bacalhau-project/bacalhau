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

package bacalhau

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var oV = &VersionOptions{
	Output: "yaml",
}

// Versions is a struct for version information
type Versions struct {
	ClientVersion *model.BuildVersionInfo `json:"clientVersion,omitempty"`
	ServerVersion *model.BuildVersionInfo `json:"serverVersion,omitempty"`
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	versionCmd.Flags().BoolVar(&oV.ClientOnly, "client", oV.ClientOnly, "If true, shows client version only (no server required).")
	versionCmd.Flags().StringVarP(&oV.Output, "output", "o", oV.Output, "One of 'yaml' or 'json'.")
}

// Options is a struct to support version command
type VersionOptions struct {
	ClientOnly bool
	Output     string

	args []string
}

// NewOptions returns initialized Options
func NewVersionOptions() *VersionOptions {
	return &VersionOptions{}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the client and server version.",
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:unparam // incorrectly suggesting unused
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		t := system.GetTracer()
		ctx, rootSpan := system.NewRootSpan(ctx, t, "cmd/bacalhau/version")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		oV.Output = strings.TrimSpace(strings.ToLower(oV.Output))

		err := oV.Validate(cmd)
		if err != nil {
			Fatal(fmt.Sprintf("Error validating version: %s\n", err), 1)
		}

		err = oV.Run(ctx, cmd)
		if err != nil {
			Fatal(fmt.Sprintf("Error running version: %s\n", err), 1)
		}

		return nil
	},
}

// Validate validates the provided options
func (oV *VersionOptions) Validate(cmd *cobra.Command) error {
	if len(oV.args) != 0 {
		return fmt.Errorf("extra arguments: %v", oV.args)
	}

	if oV.Output != "" && oV.Output != YAMLFormat && oV.Output != JSONFormat {
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
		serverVersion, err := GetAPIClient().Version(ctx)
		if err != nil {
			log.Error().Msgf("could not get server version")
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
	case YAMLFormat:
		marshaled, err := yaml.Marshal(versions)
		if err != nil {
			return err
		}
		cmd.Println(string(marshaled))
	case JSONFormat:
		marshaled, err := json.MarshalIndent(versions, "", "  ")
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
