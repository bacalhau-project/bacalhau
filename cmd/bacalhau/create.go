package bacalhau

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
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
		# Create a job using the data in job.json
		bacalhau create ./job.json

		# Create a job based on the JSON passed into stdin
		cat job.json | job create -`))

	// Set Defaults (probably a better way to do this)
	OC = NewCreateOptions()

	// For the -f flag
)

type CreateOptions struct {
	Filename    string // Filename for job (can be .json or .yaml)
	Concurrency int    // Number of concurrent jobs to run
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:    "",
		Concurrency: 1,
	}
}

func init() { //nolint:gochecknoinits
	createCmd.PersistentFlags().IntVarP(
		&OC.Concurrency, "concurrency", "c", OC.Concurrency,
		`How many nodes should run the job`,
	)
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
		ctx := context.Background()

		if len(cmdArgs) == 0 {
			_ = cmd.Usage()
			return fmt.Errorf("no filename specified")
		}
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

		jobSpec := &executor.JobSpec{}

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

		engineType, err := executor.ParseEngineType(jobSpec.EngineName)
		if err != nil {
			return err
		}

		verifierType, err := verifier.ParseVerifierType(jobSpec.VerifierName)
		if err != nil {
			return err
		}

		publisherType, err := publisher.ParsePublisherType(jobSpec.PublisherName)
		if err != nil {
			return err
		}

		jobSpec.Engine = engineType
		jobSpec.Verifier = verifierType
		jobSpec.Publisher = publisherType

		jobDeal := &executor.JobDeal{
			Concurrency: OC.Concurrency,
		}

		err = ExecuteJob(ctx,
			cm,
			cmd,
			jobSpec,
			jobDeal,
			ODR.IsLocal,
			ODR.WaitForJobToFinish,
			ODR.DockerRunDownloadFlags)

		if err != nil {
			return fmt.Errorf("error executing job: %s", err)
		}

		return nil

	},
}
