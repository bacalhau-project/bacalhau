package bacalhau

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var jobspec *executor.JobSpec
var filename string
var jobfConcurrency int
var jobfInputVolumes []string
var jobfOutputVolumes []string
var jobTags []string

func init() { // nolint:gochecknoinits
	applyCmd.PersistentFlags().StringVarP(
		&filename, "filename", "f", "",
		`Path to the job file`,
	)

	applyCmd.PersistentFlags().IntVarP(
		&jobfConcurrency, "concurrency", "c", 1,
		`How many nodes should run the job in parallel`,
	)

	applyCmd.PersistentFlags().StringSliceVarP(&jobTags,
		"labels", "l", []string{},
		`List of jobTags for the job. In the format 'a,b,c,1'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`,
	)
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Submit a job.json or job.yaml file and run it on the network",
	Args:  cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		ctx := context.Background()
		fileextension := filepath.Ext(filename)

		fileContent, err := os.Open(filename)

		if err != nil {
			return err
		}

		defer fileContent.Close()

		byteResult, err := io.ReadAll(fileContent)

		if err != nil {
			return err
		}

		if fileextension == ".json" {
			err = json.Unmarshal(byteResult, &jobspec)
			if err != nil {
				return err
			}
		}

		if fileextension == ".yaml" || fileextension == ".yml" {
			err = yaml.Unmarshal(byteResult, &jobspec)
			if err != nil {
				return err
			}
		}

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

		engineType, err := executor.ParseEngineType(jobspec.EngineName)
		if err != nil {
			cmd.Printf("Error parsing engine type: %s", err)
			return err
		}

		verifierType, err := verifier.ParseVerifierType(jobspec.VerifierName)
		if err != nil {
			cmd.Printf("Error parsing engine type: %s", err)
			return err
		}

		spec, deal, err := job.ConstructDockerJob(
			engineType,
			verifierType,
			jobspec.Resources.CPU,
			jobspec.Resources.GPU,
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

		cmd.Printf("%s\n", job.ID)
		return nil

	},
}
