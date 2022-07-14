package bacalhau

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() { // nolint:gochecknoinits // Using init with Cobra Command is ideomatic
}

type jobDescription struct {
	ID              string                       `yaml:"Id"`
	ClientID        string                       `yaml:"ClientID"`
	RequesterNodeID string                       `yaml:"RequesterNodeId"`
	Spec            jobSpecDescription           `yaml:"Spec"`
	Deal            executor.JobDeal             `yaml:"Deal"`
	State           map[string]executor.JobState `yaml:"State"`
	CreatedAt       time.Time                    `yaml:"Start Time"`
}

type jobSpecDescription struct {
	Engine     string               `yaml:"Engine"`
	Verifier   string               `yaml:"Verifier"`
	VM         jobSpecVMDescription `yaml:"VM"`
	Deployment jobDealDescription   `yaml:"Deployment"`
}

type jobSpecVMDescription struct {
	Image       string   `yaml:"Image"`
	Entrypoint  []string `yaml:"Entrypoint Command"`
	Env         []string `yaml:"Submitted Env Variables"`
	CPU         string   `yaml:"CPU Allocated"`
	Memory      string   `yaml:"Memory Allocated"`
	Inputs      []string `yaml:"Inputs"`
	Outputs     []string `yaml:"Outputs"`
	Annotations []string `yaml:"Annotations"`
}

type jobDealDescription struct {
	Concurrency   int      `yaml:"Concurrency"`
	AssignedNodes []string `yaml:"Assigned Nodes"`
}

// nolintunparam // incorrectly suggesting unused
var describeCmd = &cobra.Command{
	Use:   "describe [id]",
	Short: "Describe a job on the network",
	Long:  "Full description of a job, in yaml format. Use 'bacalhau list' to get a list of all ids. Short form and long form of the job id are accepted.", // nolint:lll
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
		// TODO: can cobra do validation on args, check this is a valid id?
		jobID := cmdArgs[0]

		job, ok, err := getAPIClient().Get(context.Background(), jobID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", jobID, err)
			return err
		}
		if !ok {
			log.Error().Msgf("No job found with ID '%s'.", jobID)
			return fmt.Errorf("no job found with ID: %s", jobID)
		}

		jobVMDesc := jobSpecVMDescription{}
		jobVMDesc.Image = job.Spec.Docker.Image
		jobVMDesc.Entrypoint = job.Spec.Docker.Entrypoint
		jobVMDesc.Env = job.Spec.Docker.Env

		jobVMDesc.CPU = job.Spec.Resources.CPU
		jobVMDesc.Memory = job.Spec.Resources.Memory

		jobSpecDesc := jobSpecDescription{}
		jobSpecDesc.Engine = executor.EngineTypes()[job.Spec.Engine].String()

		jobDealDesc := jobDealDescription{}
		jobDealDesc.Concurrency = job.Deal.Concurrency
		jobDealDesc.AssignedNodes = job.Deal.AssignedNodes

		jobSpecDesc.Verifier = job.Spec.Verifier.String()
		jobSpecDesc.VM = jobVMDesc

		jobDesc := jobDescription{}
		jobDesc.ID = job.ID
		jobDesc.ClientID = job.ClientID
		jobDesc.RequesterNodeID = job.RequesterNodeID
		jobDesc.Spec = jobSpecDesc
		jobDesc.Deal = job.Deal
		jobDesc.State = job.State
		jobDesc.CreatedAt = job.CreatedAt

		bytes, _ := yaml.Marshal(jobDesc)
		cmd.Print(string(bytes))

		return nil
	},
}
