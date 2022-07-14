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

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var o = &Options{}

// Versions is a struct for version information
type Versions struct {
	ClientVersion *executor.VersionInfo `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
	ServerVersion *executor.VersionInfo `json:"serverVersion,omitempty" yaml:"serverVersion,omitempty"`
}

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	versionCmd.Flags().BoolVar(&o.ClientOnly, "client", o.ClientOnly, "If true, shows client version only (no server required).")
	versionCmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "One of 'yaml' or 'json'.")
}

// Options is a struct to support version command
type Options struct {
	ClientOnly bool
	Output     string

	args []string
}

// nolintunparam // incorrectly suggesting unused
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the client and server version.",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
		err := o.Validate(cmd)
		if err != nil {
			log.Error().Msgf("error validating version - %s", err)
		}

		err = o.Run(cmd)
		if err != nil {
			log.Error().Msgf("error running version - %s", err)
		}

		return nil
	},
}

// Validate validates the provided options
func (o *Options) Validate(cmd *cobra.Command) error {
	if len(o.args) != 0 {
		return fmt.Errorf("extra arguments: %v", o.args)
	}

	if o.Output != "" && o.Output != "yaml" && o.Output != "json" {
		return errors.New(`--output must be 'yaml' or 'json'`)
	}

	return nil
}

// Run executes version command
func (o *Options) Run(cmd *cobra.Command) error {
	var (
		versions Versions
	)

	versions.ClientVersion = version.Get()

	if !o.ClientOnly {
		serverVersion, err := getAPIClient().Version(context.Background())
		if err != nil {
			log.Error().Msgf("could not get server version")
			return err
		}

		versions.ServerVersion = serverVersion
	}

	switch o.Output {
	case "":
		cmd.Printf("Client Version: %s\n", versions.ClientVersion.GitVersion)
		if versions.ServerVersion != nil {
			cmd.Printf("Server Version: %s\n", versions.ServerVersion.GitVersion)
		}
	case "yaml":
		marshaled, err := yaml.Marshal(versions)
		if err != nil {
			return err
		}
		cmd.Println(string(marshaled))
	case "json":
		marshaled, err := json.MarshalIndent(versions, "", "  ")
		if err != nil {
			return err
		}
		cmd.Println(string(marshaled))
	default:
		// There is a bug in the program if we hit this case.
		// However, we follow a policy of never panicking.
		return fmt.Errorf("VersionOptions were not validated: --output=%q should have been rejected", o.Output)
	}

	return nil
}

// NewOptions returns initialized Options
func NewOptions() *Options {
	return &Options{}
}
