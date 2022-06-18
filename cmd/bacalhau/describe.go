package bacalhau

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var id string

func init() {
	describeCmd.PersistentFlags().StringVarP(&id, "id", "i", "", `show all information related to a job.`)
	_ = describeCmd.MarkFlagRequired("id")
}

type jobDescription struct {
	ID string
}

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe a job on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		if len(id) == 0 {
			err := fmt.Errorf("Please submit an id with the --id flag.")
			log.Error().Msgf(err.Error())
			return err
		}

		// TODO: Create a span when Otel library comes in
		job, _, err := getAPIClient().Get(id)

		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", id, err)
			return err
		}

		jobDesc := &jobDescription{}
		jobDesc.ID = job.Id

		bytes, _ := yaml.Marshal(jobDesc)
		fmt.Println(string(bytes))

		return nil
	},
}
