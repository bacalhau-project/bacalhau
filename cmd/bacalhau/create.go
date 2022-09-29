package bacalhau

import (
	"fmt"
	"io"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"
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

		// Custom unmarshaller
		// https://stackoverflow.com/questions/70635636/unmarshaling-yaml-into-different-struct-based-off-yaml-field?rq=1
		var jwi model.JobWithInfo
		j, err := model.NewJobWithSaneProductionDefaults()
		if err != nil {
			return err
		}
		var byteResult []byte
		var rawMap map[string]interface{}

		if len(cmdArgs) == 0 {
			_ = cmd.Usage()
			return fmt.Errorf("no filename specified")
		}

		OC.Filename = cmdArgs[0]

		if OC.Filename == "-" {
			byteResult, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return errors.Wrap(err, "failed to read from stdin")
			}
		} else {
			var fileContent *os.File
			fileContent, err = os.Open(OC.Filename)

			if err != nil {
				return fmt.Errorf("could not open file '%s': %s", OC.Filename, err)
			}

			byteResult, err = io.ReadAll(fileContent)
			if err != nil {
				return errors.Wrap(err, "failed to read from file")
			}
		}

		// Do a first pass for parsing to see if it's a Job or JobWithInfo
		err = yaml.Unmarshal(byteResult, &rawMap)
		if err != nil {
			return errors.Wrap(err, "failed to read the file initial pass")
		}

		// If it's a JobWithInfo, we need to convert it to a Job
		if _, isJobWithInfo := rawMap["Job"]; isJobWithInfo {
			err = yaml.Unmarshal(byteResult, &jwi)
			if err != nil {
				log.Error().Err(err).Msg("Error creating a job from yaml. Error:")
				return err
			}
			byteResult, err = yaml.Marshal(jwi.Job)
			if err != nil {
				return errors.Wrap(err, "Error getting job out of input")
			}
		}

		err = yaml.Unmarshal(byteResult, &j)
		if err != nil {
			log.Error().Err(err).Msg("Error creating a job from input. Error:")
			return err
		}

		// the spec might use string version or proper numeric versions
		// let's convert them to the numeric version
		engineType, err := model.EnsureEngine(j.Spec.Engine, j.Spec.EngineName)
		if err != nil {
			return err
		}

		verifierType, err := model.EnsureVerifier(j.Spec.Verifier, j.Spec.VerifierName)
		if err != nil {
			return err
		}

		publisherType, err := model.EnsurePublisher(j.Spec.Publisher, j.Spec.PublisherName)
		if err != nil {
			return err
		}

		parsedInputs, err := model.EnsureStorageSpecsSourceTypes(j.Spec.Inputs)
		if err != nil {
			return err
		}

		j.Spec.Engine = engineType
		j.Spec.Verifier = verifierType
		j.Spec.Publisher = publisherType
		j.Spec.Inputs = parsedInputs

		err = ExecuteJob(ctx,
			cm,
			cmd,
			j,
			OC.RunTimeSettings,
			OC.DownloadFlags,
		)

		if err != nil {
			return fmt.Errorf("error executing job: %s", err)
		}

		return nil

	},
}
