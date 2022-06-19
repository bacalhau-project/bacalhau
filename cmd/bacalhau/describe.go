package bacalhau

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var id string

func init() {
	describeCmd.PersistentFlags().StringVarP(&id, "id", "i", "", `show all information related to a job.`)
}

type jobDescription struct {
	ID        string                    `yaml:"Id"`
	Owner     string                    `yaml:"Owner"`
	Spec      types.JobSpec             `yaml:"Spec"`
	Deal      types.JobDeal             `yaml:"Deal"`
	State     map[string]types.JobState `yaml:"State"`
	CreatedAt time.Time                 `yaml:"Start Time"`
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

		job, _, err := getAPIClient().Get(context.Background(), id)
		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", id, err)
			return err
		}

		jobDesc := &jobDescription{}
		jobDesc.ID = job.Id
		jobDesc.Owner = job.Owner
		jobDesc.Spec = *job.Spec
		jobDesc.Deal = *job.Deal
		jobDesc.State = job.State

		bytes, _ := yaml.Marshal(jobDesc)
		fmt.Println(string(bytes))

		return nil
	},
}
