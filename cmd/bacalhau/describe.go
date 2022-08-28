package bacalhau

import (
	"context"
	"sort"
	"time"

	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	//nolint:lll // Documentation
	describeLong = templates.LongDesc(i18n.T(`
		Full description of a job, in yaml format. Use 'bacalhau list' to get a list of all ids. Short form and long form of the job id are accepted.
`))
	//nolint:lll // Documentation
	describeExample = templates.Examples(i18n.T(`
		# Describe a job with the full ID
		bacalhau describe e3f8c209-d683-4a41-b840-f09b88d087b9

		# Describe a job with the a shortened ID
		bacalhau describe 47805f5c
`))

	// Set Defaults (probably a better way to do this)
	OD = NewDescribeOptions()

	// For the -f flag
)

type DescribeOptions struct {
	Filename    string // Filename for job (can be .json or .yaml)
	Concurrency int    // Number of concurrent jobs to run
}

func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{}
}
func init() { //nolint:gochecknoinits // Using init with Cobra Command is ideomatic
}

type eventDescription struct {
	Event       string `yaml:"Event"`
	Time        string `yaml:"Time"`
	Concurrency int    `yaml:"Concurrency"`
	SourceNode  string `yaml:"SourceNode"`
	TargetNode  string `yaml:"TargetNode"`
	Status      string `yaml:"Status"`
}

type localEventDescription struct {
	Event      string `yaml:"Event"`
	TargetNode string `yaml:"TargetNode"`
}

type shardNodeStateDescription struct {
	Node     string `yaml:"Node"`
	State    string `yaml:"State"`
	Status   string `yaml:"Status"`
	ResultID string `yaml:"ResultID"`
}

type shardStateDescription struct {
	ShardIndex int                         `yaml:"ShardIndex"`
	Nodes      []shardNodeStateDescription `yaml:"Nodes"`
}

type jobDescription struct {
	ID              string                  `yaml:"Id"`
	ClientID        string                  `yaml:"ClientID"`
	RequesterNodeID string                  `yaml:"RequesterNodeId"`
	Spec            jobSpecDescription      `yaml:"Spec"`
	Deal            model.JobDeal           `yaml:"Deal"`
	Shards          []shardStateDescription `yaml:"Shards"`
	CreatedAt       time.Time               `yaml:"Start Time"`
	Events          []eventDescription      `yaml:"Events"`
	LocalEvents     []localEventDescription `yaml:"LocalEvents"`
}

type jobSpecDescription struct {
	Engine     string                   `yaml:"Engine"`
	Verifier   string                   `yaml:"Verifier"`
	Docker     jobSpecDockerDescription `yaml:"Docker"`
	Deployment jobDealDescription       `yaml:"Deployment"`
}

type jobSpecDockerDescription struct {
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

var describeCmd = &cobra.Command{
	Use:     "describe [id]",
	Short:   "Describe a job on the network",
	Long:    describeLong,
	Example: describeExample,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
		inputJobID := cmdArgs[0]

		j, ok, err := getAPIClient().Get(context.Background(), cmdArgs[0])

		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", inputJobID, err)
			return err
		}

		if !ok {
			cmd.Printf("No job ID found matching ID: %s", inputJobID)
			return nil
		}

		jobState, err := getAPIClient().GetJobState(context.Background(), j.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job states '%s': %s", j.ID, err)
			return err
		}

		jobEvents, err := getAPIClient().GetEvents(context.Background(), j.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job events '%s': %s", j.ID, err)
			return err
		}

		localEvents, err := getAPIClient().GetLocalEvents(context.Background(), j.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job events '%s': %s", j.ID, err)
			return err
		}

		jobDockerDesc := jobSpecDockerDescription{}
		jobDockerDesc.Image = j.Spec.Docker.Image
		jobDockerDesc.Entrypoint = j.Spec.Docker.Entrypoint
		jobDockerDesc.Env = j.Spec.Docker.Env

		jobDockerDesc.CPU = j.Spec.Resources.CPU
		jobDockerDesc.Memory = j.Spec.Resources.Memory

		jobSpecDesc := jobSpecDescription{}
		jobSpecDesc.Engine = j.Spec.Engine.String()

		jobDealDesc := jobDealDescription{}
		jobDealDesc.Concurrency = j.Deal.Concurrency

		jobSpecDesc.Verifier = j.Spec.Verifier.String()
		jobSpecDesc.Docker = jobDockerDesc

		jobDesc := jobDescription{}
		jobDesc.ID = j.ID
		jobDesc.ClientID = j.ClientID
		jobDesc.RequesterNodeID = j.RequesterNodeID
		jobDesc.Spec = jobSpecDesc
		jobDesc.Deal = j.Deal
		jobDesc.CreatedAt = j.CreatedAt
		jobDesc.Events = []eventDescription{}

		shardDescriptions := map[int]shardStateDescription{}

		for _, shard := range jobutils.FlattenShardStates(jobState) {
			shardDescription, ok := shardDescriptions[shard.ShardIndex]
			if !ok {
				shardDescription = shardStateDescription{
					ShardIndex: shard.ShardIndex,
					Nodes:      []shardNodeStateDescription{},
				}
			}
			shardDescription.Nodes = append(shardDescription.Nodes, shardNodeStateDescription{
				Node:     shard.NodeID,
				State:    shard.State.String(),
				Status:   shard.Status,
				ResultID: string(shard.VerificationProposal),
			})
			shardDescriptions[shard.ShardIndex] = shardDescription
		}

		shardIndexes := []int{}
		for shardIndex := range shardDescriptions {
			shardIndexes = append(shardIndexes, shardIndex)
		}

		sort.Ints(shardIndexes)

		finalDescriptions := []shardStateDescription{}

		for _, shardIndex := range shardIndexes {
			finalDescriptions = append(finalDescriptions, shardDescriptions[shardIndex])
		}

		jobDesc.Shards = finalDescriptions

		for _, event := range jobEvents {
			jobDesc.Events = append(jobDesc.Events, eventDescription{
				Event:       event.EventName.String(),
				Status:      event.Status,
				Time:        event.EventTime.String(),
				Concurrency: event.JobDeal.Concurrency,
				SourceNode:  event.SourceNodeID,
				TargetNode:  event.TargetNodeID,
			})
		}

		jobDesc.LocalEvents = []localEventDescription{}
		for _, event := range localEvents {
			jobDesc.LocalEvents = append(jobDesc.LocalEvents, localEventDescription{
				Event:      event.EventName.String(),
				TargetNode: event.TargetNodeID,
			})
		}

		bytes, err := yaml.Marshal(jobDesc)
		if err != nil {
			log.Error().Msgf("Failure marshaling job description '%s': %s", j.ID, err)
			return err
		}

		cmd.Print(string(bytes))

		return nil
	},
}
