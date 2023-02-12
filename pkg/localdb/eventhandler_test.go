//go:build unit || !integration

package localdb

import (
	"reflect"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestEventHandlerSuite(t *testing.T) {
	suite.Run(t, new(EventHandlerSuite))
}

type EventHandlerSuite struct {
	suite.Suite
}

// Before each test
func (suite *EventHandlerSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
}

func (suite *EventHandlerSuite) TestRun_ConstructJobFromEvent() {
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
