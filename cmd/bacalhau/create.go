package bacalhau

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
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
		bacalhau create -f ./job.json

		# Create a job based on the JSON passed into stdin
		cat job.json | job create -f -`))

	// Set Defaults (probably a better way to do this)
	OC = NewDockerRunOptions()

	// For the -f flag
	filename = ""
)

// DockerRunOptions declares the arguments accepted by the `docker run` command
type CreateOptions struct {
	Filename        string   // Filename for job (can be .json or .yaml)
	Concurrency   	int      // Number of concurrent jobs to run
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:			"",
		Concurrency: el        1,
	}
}


func init() { //nolint:gochecknoinits
	createCmd.PersistentFlags().StringVarP(
		&filename, "filename", "f", "",
		`Path to the job file`,
	)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a job using a json or yaml file.",
	Long:  "Create a job using a json or yaml file.",
	Args:  cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { //nolint:unparam // incorrect that cmd is unused.
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

		jobSpec := &executor.JobSpec{}

		if fileextension == ".json" {
			err = json.Unmarshal(byteResult, &jobSpec)
			if err != nil {
				return err
			}
		} else if fileextension == ".yaml" || fileextension == ".yml" {
			err = yaml.Unmarshal(byteResult, &jobSpec)
			if err != nil {
				return err
			}
		}

		job, err := getAPIClient().Submit(ctx, jobSpec, deal, nil)
		if err != nil {
			return err
		}

		cmd.Printf("%s\n", job.ID)
		return nil

	},
}
