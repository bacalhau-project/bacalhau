//go:build unit || !integration

package job

import (
	"reflect"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestJobFactorySuite(t *testing.T) {
	suite.Run(t, new(JobFactorySuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type JobFactorySuite struct {
	suite.Suite
}

// Before each test
func (suite *JobFactorySuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
}

func (suite *JobFactorySuite) TestRun_DockerJobOutputs() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	for range tests {
		type (
			OutputVolumes struct {
				name string
				path string
			}
		)

		testCids := []struct {
			outputVolumes []OutputVolumes
			correctLength int
			err           string
		}{
			{outputVolumes: []OutputVolumes{{name: "", path: ""}}, correctLength: 0, err: "invalid output volume"}, // Flag not provided
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME", path: "/outputs"}}, correctLength: 1, err: ""},
			{outputVolumes: []OutputVolumes{{name: "APPLE_1", path: "/apple"}, {name: "APPLE_2", path: "/apple"}}, correctLength: 2, err: ""},
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME", path: "/outputs"}, {name: "OUTPUT_NAME_1", path: "/outputs_1"}}, correctLength: 2, err: ""},     // Two outputs, one default (and dupe), one not
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME_1", path: "/outputs_1"}}, correctLength: 2, err: ""},                                              // Correct output flag
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME_2", path: "/outputs_2"}, {name: "OUTPUT_NAME_3", path: "/outputs_3"}}, correctLength: 3, err: ""}, // 2 correct output flags
			{outputVolumes: []OutputVolumes{{name: "OUTPUT_NAME_4", path: ""}}, correctLength: 0, err: "invalid output volume"},                                   // OV requested but no path (should error)
			{outputVolumes: []OutputVolumes{{name: "", path: "/outputs_4"}}, correctLength: 0, err: "invalid output volume"},                                      // OV requested but no name (should error)
		}

		for _, tcids := range testCids {
			func() {
				outputVolumes := []string{}
				for _, tcidOV := range tcids.outputVolumes {
					outputVolumes = append(outputVolumes, strings.Join([]string{tcidOV.name, tcidOV.path}, ":"))
				}

				j, err := ConstructDockerJob( //nolint:funlen
					model.APIVersionLatest(),
					model.EngineNoop,
					model.VerifierNoop,
					model.PublisherNoop,
					"1",               // cpu
					"1",               // memory
					"0",               // gpu
					model.NetworkNone, // networking
					[]string{},        // domains
					[]string{},        // input urls
					[]string{},        // input volumes
					outputVolumes,
					[]string{}, // env
					[]string{}, // entrypoint
					"",         // image
					1,          // concurrency
					0,          // confidence
					0,          // min bids
					300,        // timeout
					[]string{}, // annotations
					"",         // node selector
					"",         // working dir
					"",         // sharding base path
					"",         // sharding glob pattern
					1,          // sharding batch size
					true,       // do not track
				)

				if tcids.err != "" {
					require.Error(suite.T(), err, "No error received, but error expected - %+v", tcids)
					require.Contains(suite.T(), err.Error(), tcids.err, "Error does not contain expected - %+v - %+v", tcids, err)
				} else {
					require.NoError(suite.T(), err, "Error in creating spec - %+v", tcids)
					require.Equal(suite.T(), len(j.Spec.Outputs),
						tcids.correctLength,
						"Length of deal outputs (%d) not the same as expected (%d). %+v",
						len(j.Spec.Outputs),
						tcids.correctLength,
						tcids.outputVolumes,
					)
				}
			}()
		}
	}
}

func (suite *JobFactorySuite) TestRun_ConstructJobFromEvent() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1},
	}

	// Expect a Job create form an event to have all of the following fields
	requiredJobFields := []string{
		"APIVersion",
		"Metadata",
		"Spec",
		"Status",
	}

	for range tests {

		testEvents := []struct {
			jobEvent      model.JobEvent
			err           string
			missingFields []string
		}{
			{
				model.JobEvent{
					APIVersion:      model.APIVersionLatest().String(),
					JobID:           "1111",
					ClientID:        "2222",
					SourceNodeID:    "3333",
					SenderPublicKey: []byte("sender-pub-key"),
					Spec: model.Spec{
						Engine:    model.EngineNoop,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherNoop,
					},
					Deal: model.Deal{
						Concurrency: 1,
					},
					JobExecutionPlan: model.JobExecutionPlan{
						TotalShards: 1,
					},
				},
				"",
				[]string{},
			},
			{
				model.JobEvent{
					JobID:           "job-id1",
					ClientID:        "test-client-id",
					SourceNodeID:    "test-src-node-id",
					SenderPublicKey: []byte("test-sender-pub-key"),
					Spec: model.Spec{
						Engine:    model.EngineNoop,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherNoop,
					},
					Deal: model.Deal{
						Concurrency: 1,
					},
					JobExecutionPlan: model.JobExecutionPlan{
						TotalShards: 1,
					},
				},
				"Missing APIVersion",
				[]string{"APIVersion"},
			},
			{
				model.JobEvent{
					APIVersion:      model.APIVersionLatest().String(),
					ClientID:        "2222",
					SourceNodeID:    "3333",
					SenderPublicKey: []byte("sender-pub-key"),
					Spec: model.Spec{
						Engine:    model.EngineNoop,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherNoop,
					},
					Deal: model.Deal{
						Concurrency: 1,
					},
					JobExecutionPlan: model.JobExecutionPlan{
						TotalShards: 1,
					},
				},
				"Missing JobID",
				[]string{"JobID"},
			},
			{
				model.JobEvent{
					APIVersion:      model.APIVersionLatest().String(),
					SourceNodeID:    "3333",
					SenderPublicKey: []byte("sender-pub-key"),
					Spec: model.Spec{
						Engine:    model.EngineNoop,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherNoop,
					},
					Deal: model.Deal{
						Concurrency: 1,
					},
					JobExecutionPlan: model.JobExecutionPlan{
						TotalShards: 1,
					},
				},
				"Missing JobID",
				[]string{"JobID"},
			},
		}

		for _, tevent := range testEvents {
			func() {
				j := ConstructJobFromEvent(tevent.jobEvent)

				if tevent.err != "" {
					for _, missingField := range tevent.missingFields {
						if missingField == "APIVersion" {
							require.Empty(suite.T(), j.APIVersion, "APIVersion should be empty - %+v", j)
						} else if missingField == "JobID" {
							require.Empty(suite.T(), j.Metadata.ID, "JobID should be empty - %+v", j)
						} else if missingField == "ClientID" {
							require.Empty(suite.T(), j.Metadata.ClientID, "ClientID should be empty - %+v", j)
						}
					}
				} else {
					// Expect required fields to exist
					for _, field := range requiredJobFields {
						require.False(suite.T(), reflect.DeepEqual(reflect.ValueOf(j).Elem().FieldByName(field), reflect.Value{}), "Field %s not found in job - %+v", field, j)
					}

					// check if fields match
					require.Equal(suite.T(), j.APIVersion, tevent.jobEvent.APIVersion, "Job does not contain expected APIVersion value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Metadata.ID, tevent.jobEvent.JobID, "Job does not contain expected JobID value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Metadata.ClientID, tevent.jobEvent.ClientID, "Job does not contain expected ClientID value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Spec.Engine, tevent.jobEvent.Spec.Engine, "Job does not contain expected Spec.Engine value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Spec.Verifier, tevent.jobEvent.Spec.Verifier, "Job does not contain expected Spec.Verifier value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Spec.Publisher, tevent.jobEvent.Spec.Publisher, "Job does not contain expected Spec.Publisher value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Spec.Deal.Concurrency, tevent.jobEvent.Deal.Concurrency, "Job does not contain expected Spec.Deal.Concurrency value - %+v - %+v", tevent.jobEvent, j)
					require.Equal(suite.T(), j.Spec.ExecutionPlan.TotalShards, tevent.jobEvent.JobExecutionPlan.TotalShards, "Job does not contain expected Spec.ExecutionPlan.TotalShards value - %+v - %+v", tevent.jobEvent, j)
				}
			}()
		}
	}
}
