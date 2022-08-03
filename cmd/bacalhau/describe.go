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

type jobShardDescription struct {
	ShardIndex int `yaml:"ShardIndex"`
}

type jobDescription struct {
	ID              string                  `yaml:"Id"`
	ClientID        string                  `yaml:"ClientID"`
	RequesterNodeID string                  `yaml:"RequesterNodeId"`
	Spec            jobSpecDescription      `yaml:"Spec"`
	Deal            executor.JobDeal        `yaml:"Deal"`
	State           executor.JobState       `yaml:"State"`
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

// nolintunparam // incorrectly suggesting unused
var describeCmd = &cobra.Command{
	Use:   "describe [id]",
	Short: "Describe a job on the network",
	Long:  "Full description of a job, in yaml format. Use 'bacalhau list' to get a list of all ids. Short form and long form of the job id are accepted.", // nolint:lll
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
		inputJobID := cmdArgs[0]

		job, ok, err := getAPIClient().Get(context.Background(), cmdArgs[0])

		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", inputJobID, err)
			return err
		}

		if !ok {
			err = fmt.Errorf("no job found with ID: %s", inputJobID)
			log.Error().Msgf(err.Error())
			return err
		}

		state, err := getAPIClient().GetJobState(context.Background(), job.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job states '%s': %s", job.ID, err)
			return err
		}

		events, err := getAPIClient().GetEvents(context.Background(), job.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job events '%s': %s", job.ID, err)
			return err
		}

		localEvents, err := getAPIClient().GetLocalEvents(context.Background(), job.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job events '%s': %s", job.ID, err)
			return err
		}

		jobDockerDesc := jobSpecDockerDescription{}
		jobDockerDesc.Image = job.Spec.Docker.Image
		jobDockerDesc.Entrypoint = job.Spec.Docker.Entrypoint
		jobDockerDesc.Env = job.Spec.Docker.Env

		jobDockerDesc.CPU = job.Spec.Resources.CPU
		jobDockerDesc.Memory = job.Spec.Resources.Memory

		jobSpecDesc := jobSpecDescription{}
		jobSpecDesc.Engine = job.Spec.Engine.String()

		jobDealDesc := jobDealDescription{}
		jobDealDesc.Concurrency = job.Deal.Concurrency

		jobSpecDesc.Verifier = job.Spec.Verifier.String()
		jobSpecDesc.Docker = jobDockerDesc

		jobDesc := jobDescription{}
		jobDesc.ID = job.ID
		jobDesc.ClientID = job.ClientID
		jobDesc.RequesterNodeID = job.RequesterNodeID
		jobDesc.Spec = jobSpecDesc
		jobDesc.Deal = job.Deal
		jobDesc.State = state
		jobDesc.CreatedAt = job.CreatedAt
		jobDesc.Events = []eventDescription{}
		for _, event := range events {
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
			log.Error().Msgf("Failure marshaling job description '%s': %s", job.ID, err)
			return err
		}

		cmd.Print(string(bytes))

		return nil
	},
}
