//go:build unit || !integration

package transport

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// NATSTransportConfigSuite defines the suite for testing NATSTransportConfig
type NATSTransportConfigSuite struct {
	suite.Suite
}

// TestValidate tests the Validate method
func (suite *NATSTransportConfigSuite) TestValidate() {
	tests := []struct {
		name           string
		config         NATSTransportConfig
		expectedErrors []string
	}{
		{
			name: "Valid Config",
			config: NATSTransportConfig{
				NodeID:        "node1",
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: nil,
		},
		{
			name: "Missing NodeID",
			config: NATSTransportConfig{
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: []string{"missing node ID"},
		},
		{
			name: "NodeID Contains Space",
			config: NATSTransportConfig{
				NodeID:        "node 1",
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: []string{"node ID contains a space"},
		},
		{
			name: "NodeID Contains Null Character",
			config: NATSTransportConfig{
				NodeID:        "node\x00ID",
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: []string{"node ID contains a null character"},
		},
		{
			name: "NodeID Contains > Character",
			config: NATSTransportConfig{
				NodeID:        "node>ID",
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: []string{"contains one or more reserved character"},
		},
		{
			name: "NodeID Contains . Character",
			config: NATSTransportConfig{
				NodeID:        "node.ID",
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: []string{"contains one or more reserved character"},
		},
		{
			name: "NodeID Contains * Character",
			config: NATSTransportConfig{
				NodeID:        "node*ID",
				Orchestrators: []string{"orch1", "orch2"},
			},
			expectedErrors: []string{"contains one or more reserved character"},
		},
		{
			name: "Missing Orchestrators in Non-Requester Node",
			config: NATSTransportConfig{
				NodeID: "node2",
			},
			expectedErrors: []string{"missing orchestrators"},
		},
		{
			name: "Missing Port in Requester Node",
			config: NATSTransportConfig{
				NodeID:          "node3",
				IsRequesterNode: true,
			},
			expectedErrors: []string{"port 0 must be greater than zero"},
		},
		{
			name: "Invalid Cluster Port in Requester Node with Cluster Config",
			config: NATSTransportConfig{
				NodeID:                   "node4",
				Port:                     4222,
				IsRequesterNode:          true,
				ClusterName:              "cluster2",
				ClusterPort:              -1, // Invalid cluster port
				ClusterAdvertisedAddress: "localhost",
				ClusterPeers:             []string{"node5"},
			},
			expectedErrors: []string{"cluster port -1 must be greater than zero"},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.config.Validate()
			if len(tt.expectedErrors) == 0 {
				suite.NoError(err)
			} else {
				suite.Error(err)
				for _, errMsg := range tt.expectedErrors {
					suite.Contains(err.Error(), errMsg)
				}
			}
		})
	}
}

func TestNATSTransportConfigSuite(t *testing.T) {
	suite.Run(t, new(NATSTransportConfigSuite))
}
