package bacalhau

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var id string

func init() { // nolint:gochecknoinits // Using init with Cobra Command is ideomatic
}

type jobDescription struct {
	ID        string                        `yaml:"Id"`
	Owner     string                        `yaml:"Owner"`
	Spec      jobSpecDescription            `yaml:"Spec"`
	Deal      executor.JobDeal              `yaml:"Deal"`
	State     map[string]*executor.JobState `yaml:"State"`
	CreatedAt time.Time                     `yaml:"Start Time"`
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
		job, _, err := getAPIClient().Get(context.Background(), id)
		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", id, err)
			return err
		}

		jobVMDesc := &jobSpecVMDescription{}
		jobVMDesc.Image = job.Spec.Docker.Image
		jobVMDesc.Entrypoint = job.Spec.Docker.Entrypoint
		jobVMDesc.Env = job.Spec.Docker.Env

		jobVMDesc.CPU = job.Spec.Resources.CPU
		jobVMDesc.Memory = job.Spec.Resources.Memory

		jobSpecDesc := &jobSpecDescription{}
		jobSpecDesc.Engine = executor.EngineTypes()[job.Spec.Engine].String()

		jobDealDesc := &jobDealDescription{}
		jobDealDesc.Concurrency = job.Deal.Concurrency
		jobDealDesc.AssignedNodes = job.Deal.AssignedNodes

		// TODO: Ugh, do we have to special case this?
		jobSpecDesc.Verifier = verifier.VerifierTypes()[job.Spec.Verifier-2].String()
		jobSpecDesc.VM = *jobVMDesc

		jobDesc := &jobDescription{}
		jobDesc.ID = job.ID
		jobDesc.Owner = job.Owner
		jobDesc.Spec = *jobSpecDesc
		jobDesc.Deal = *job.Deal
		jobDesc.State = job.State
		jobDesc.CreatedAt = job.CreatedAt

		bytes, _ := yaml.Marshal(jobDesc)
		cmd.Print(string(bytes))

		return nil
	},
}
