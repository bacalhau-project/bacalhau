package bacalhau

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	createLong = templates.LongDesc(i18n.T(`
		Create a job from a file or from stdin.

		JSON and YAML formats are accepted.
	`))
	//nolint:lll // Documentation
	createExample = templates.Examples(i18n.T(`
		# Create a job using the data in job.yaml
		bacalhau create ./job.yaml

		# Create a new job from an already executed job
		bacalhau describe 6e51df50 | bacalhau create -`))

	// Set Defaults (probably a better way to do this)
	OC = NewCreateOptions()

	// For the -f flag
)

type CreateOptions struct {
	Filename        string                    // Filename for job (can be .json or .yaml)
	Concurrency     int                       // Number of concurrent jobs to run
	Confidence      int                       // Minimum number of nodes that must agree on a verification result
	RunTimeSettings RunTimeSettings           // Run time settings for execution (e.g. wait, get, etc after submission)
	DownloadFlags   ipfs.IPFSDownloadSettings // Settings for running Download
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:        "",
		Concurrency:     1,
		Confidence:      0,
		DownloadFlags:   *ipfs.NewIPFSDownloadSettings(),
		RunTimeSettings: *NewRunTimeSettings(),
	}
}

func init() { //nolint:gochecknoinits
	createCmd.PersistentFlags().IntVarP(
		&OC.Concurrency, "concurrency", "c", OC.Concurrency,
		`How many nodes should run the job`,
	)
	createCmd.PersistentFlags().IntVar(
		&OC.Confidence, "confidence", OC.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)

	setupDownloadFlags(createCmd, &OC.DownloadFlags)
	setupRunTimeFlags(createCmd, &OC.RunTimeSettings)
}

var createCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a job using a json or yaml file.",
	Long:    createLong,
	Example: createExample,
	Args:    cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { //nolint:unparam // incorrect that cmd is unused.
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		t := system.GetTracer()
		ctx, rootSpan := system.NewRootSpan(ctx, t, "cmd/bacalhau/create")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		jobSpec := &model.JobSpec{}

		if cmdArgs[0] == "-" {
			var job jobDescription

			byteResult, _ := io.ReadAll(os.Stdin)

			err := yaml.Unmarshal(byteResult, &job)
			if err != nil {
				return fmt.Errorf("error reading from stdin : %s", err)
			}
			bytes, err := yaml.Marshal(job.Spec)
			if err != nil {
				log.Error().Msgf("Failure marshaling job description : %s", err)
				return err
			}
			err = yaml.Unmarshal(bytes, &jobSpec)
			if err != nil {
				return fmt.Errorf("error reading josbpec from stdin : %s", err)
			}

		}
		if len(cmdArgs) == 0 {
			_ = cmd.Usage()
			return fmt.Errorf("no filename specified")
		}
		if cmdArgs[0] != "-" {
			OC.Filename = cmdArgs[0]

			fileextension := filepath.Ext(OC.Filename)
			fileContent, err := os.Open(OC.Filename)

			if err != nil {
				return fmt.Errorf("could not open file '%s': %s", OC.Filename, err)
			}

			byteResult, err := io.ReadAll(fileContent)

			if err != nil {
				return err
			}

			if fileextension == ".json" {
				err = json.Unmarshal(byteResult, &jobSpec)
				if err != nil {
					return fmt.Errorf("error reading json file '%s': %s", OC.Filename, err)
				}
			} else if fileextension == ".yaml" || fileextension == ".yml" {
				err = yaml.Unmarshal(byteResult, &jobSpec)
				if err != nil {
					return fmt.Errorf("error reading yaml file '%s': %s", OC.Filename, err)
				}
			} else {
				return fmt.Errorf("file '%s' must be a .json or .yaml/.yml file", OC.Filename)
			}
		}

		// the spec might use string version or proper numeric versions
		// let's convert them to the numeric version
		engineType, err := model.EnsureEngineType(jobSpec.Engine, jobSpec.EngineName)
		if err != nil {
			return err
		}

		verifierType, err := model.EnsureVerifierType(jobSpec.Verifier, jobSpec.VerifierName)
		if err != nil {
			return err
		}

		publisherType, err := model.EnsurePublisherType(jobSpec.Publisher, jobSpec.PublisherName)
		if err != nil {
			return err
		}

		parsedInputs, err := model.EnsureStorageSpecsSourceTypes(jobSpec.Inputs)
		if err != nil {
			return err
		}

		jobSpec.Engine = engineType
		jobSpec.Verifier = verifierType
		jobSpec.Publisher = publisherType
		jobSpec.Inputs = parsedInputs

		jobDeal := &model.JobDeal{
			Concurrency: OC.Concurrency,
			Confidence:  OC.Confidence,
		}

		err = ExecuteJob(ctx,
			cm,
			cmd,
			jobSpec,
			jobDeal,
			OC.RunTimeSettings,
			OC.DownloadFlags,
		)

		if err != nil {
			return fmt.Errorf("error executing job: %s", err)
		}

		return nil

	},
}
