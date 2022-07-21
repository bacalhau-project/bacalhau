package bacalhau

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var jobspec *executor.JobSpec
var filename string
var jobfConcurrency int
var jobfInputVolumes []string
var jobfOutputVolumes []string
var jobTags []string

func init() {

	applyCmd.PersistentFlags().StringVarP(
		&filename, "filename", "f", "",
		`Whats the path of the job file`,
	)

	applyCmd.PersistentFlags().IntVarP(
		&jobfConcurrency, "concurrency", "c", 1,
		`How many nodes should run the same job parallely`,
	)

	applyCmd.PersistentFlags().StringSliceVarP(&jobTags,
		"labels", "l", []string{},
		`List of jobTags for the job. In the format 'a,b,c,1'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`,
	)

}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Submit a job.json file and run it on the network",
	Args:  cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		ctx := context.Background()
		fileextension := filepath.Ext(filename)

		fileContent, err := os.Open(filename)

		if err != nil {
			return err
		}

		defer fileContent.Close()

		byteResult, _ := ioutil.ReadAll(fileContent)

		if fileextension == ".json" {
			json.Unmarshal(byteResult, &jobspec)
		}

		if fileextension == ".yaml" || fileextension == "yml" {
			yaml.Unmarshal(byteResult, &jobspec)
		}

		jobfEngine := jobspec.Engine.String()

		jobfVerifier := jobspec.Verifier.String()

		jobImage := jobspec.Docker.Image

		jobEntrypoint := jobspec.Docker.Entrypoint

		if len(jobspec.Inputs) != 0 {
			for _, jobspecsInputs := range jobspec.Inputs {
				is := jobspecsInputs.Cid + ":" + jobspecsInputs.Path
				jobfInputVolumes = append(jobfInputVolumes, is)

			}
		}
		if len(jobspec.Outputs) != 0 {
			for _, jobspecsOutputs := range jobspec.Outputs {
				is := jobspecsOutputs.Name + ":" + jobspecsOutputs.Path
				jobfOutputVolumes = append(jobfOutputVolumes, is)

			}
		}

		engineType, err := executor.ParseEngineType(jobfEngine)
		if err != nil {
			cmd.Print("error here")
			return err
		}

		verifierType, err := verifier.ParseVerifierType(jobfVerifier)
		if err != nil {
			return err
		}

		shells := strings.Split(`/bin/sh
		/bin/bash
		/usr/bin/bash
		/bin/rbash
		/usr/bin/rbash
		/usr/bin/sh
		/bin/dash
		/usr/bin/dash
		/usr/bin/tmux
		/usr/bin/screen
		/bin/zsh
		/usr/bin/zsh`, "/n")

		containsGlob := false
		for _, entrypointArg := range jobEntrypoint {
			if strings.ContainsAny(entrypointArg, "*") {
				containsGlob = true
			}
		}

		if containsGlob {
			for _, shell := range shells {
				if strings.Index(strings.TrimSpace(jobEntrypoint[0]), shell) == 0 {
					containsGlob = false
					break
				}
			}
			if containsGlob {
				log.Warn().Msgf("We could not help but notice your command contains a glob, but does not start with a shell. This is almost certainly not going to work. To use globs, you must start your command with a shell (e.g. /bin/bash <your command>).") // nolint:lll // error message
			}
		}

		spec, deal, err := job.ConstructDockerJob(
			engineType,
			verifierType,
			jobspec.Resources.CPU,
			jobspec.Resources.Memory,
			jobfInputVolumes,
			jobfOutputVolumes,
			jobspec.Docker.Env,
			jobEntrypoint,
			jobImage,
			jobfConcurrency,
			jobTags,
		)
		if err != nil {
			return err
		}

		if !skipSyntaxChecking {
			err = system.CheckBashSyntax(jobEntrypoint)
			if err != nil {
				return err
			}
		}

		job, err := getAPIClient().Submit(ctx, spec, deal, nil)
		if err != nil {
			return err
		}

		cmd.Printf("spec %#v, \n deal %v", spec, deal)
		cmd.Printf("cmdArgs %v", cmdArgs)
		cmd.Printf("%s\n", job.ID)
		return nil

	},
}
