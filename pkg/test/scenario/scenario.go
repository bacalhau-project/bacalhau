/*
Package scenario provides a high-level testing framework for running Bacalhau
jobs in different configurations and making assertions against the results.

The unit of measure is the `Scenario` which decsribes a Bacalhau network, a job
to be submitted to it, and a set of checks to exercise that the job was executed
as expected.

Scenarios can be used in standalone way (see `pkg/test/executor/test_runner.go`)
or using the provided `ScenarioRunner` can be used.

As well as executing jobs against real executors, a Scenario can instead use the
NoopExecutor to implement a mocked out job. This makes is easier to test network
features without needing to invent a real job.
*/
package scenario

import (
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

// A Scenario represents a repeatable test case of submitting a job against a
// Bacalhau network.
//
// The Scenario defines:
//
//   - the topology and configuration of network that is required
//   - the job that will be submitted
//   - the conditions for the job to be considered successful or not
//
// Most of the fields in a Scenario are optional and sensible defaults will be
// used if they are not present. All that is really required is the Spec which
// details what job to run.
type Scenario struct {
	// An optional set of configuration options that define the network of nodes
	// that the job will be run against. When unspecified, the Stack will
	// consist of one node with requestor and compute nodes set up according to
	// their default configuration, and without a Noop executor.
	Stack *StackConfig

	// Setup routines which define data available to the job.
	// If nil, no storage will be set up.
	Inputs SetupStorage

	// Output volumes that must be available to the job. If nil, no output
	// volumes will be attached to the job.
	Outputs []model.StorageSpec

	// The job specification
	Spec model.Spec

	// The job deal. If nil, concurrency will default to 1.
	Deal model.Deal

	// A function that will assert submitJob response is as expected.
	// if nil, will use SubmitJobSuccess by default.
	SubmitChecker CheckSubmitResponse

	// A function that will decide whether or not the job was successful. If
	// nil, no check will be performed on job outputs.
	ResultsChecker CheckResults

	// A set of checkers that will decide when the job has completed, and maybe
	// whether it was successful or not. If empty, the job will not be waited
	// for once it has been submitted.
	JobCheckers []job.CheckStatesFunction

	//olgibbons: delete this if it doesn't work:
	CLIParameters []string
}

// All the information that is needed to uniquely define a devstack.
type StackConfig struct {
	*devstack.DevStackOptions
	node.ComputeConfig
	node.RequesterConfig
	noop.ExecutorConfig
}
