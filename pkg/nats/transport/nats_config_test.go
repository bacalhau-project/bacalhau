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
				AuthSecret:    "sekret",
			},
			expectedErrors: nil,
		},
		{
			name: "Missing NodeID",
			config: NATSTransportConfig{
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
			},
			expectedErrors: []string{"missing node ID"},
		},
		{
			name: "NodeID Contains Space",
			config: NATSTransportConfig{
				NodeID:        "node 1",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
			},
			expectedErrors: []string{"node ID cannot contain spaces"},
		},
		{
			name: "NodeID Contains Null Character",
			config: NATSTransportConfig{
				NodeID:        "node\x00ID",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
			},
			expectedErrors: []string{"node ID cannot contain null characters"},
		},
		{
			name: "NodeID Contains > Character",
			config: NATSTransportConfig{
				NodeID:        "node>ID",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
			},
			expectedErrors: []string{"node ID cannot contain any of the following characters:"},
		},
		{
			name: "NodeID Contains . Character",
			config: NATSTransportConfig{
				NodeID:        "node.ID",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
			},
			expectedErrors: []string{"node ID cannot contain any of the following characters:"},
		},
		{
			name: "NodeID Contains * Character",
			config: NATSTransportConfig{
				NodeID:        "node*ID",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
			},
			expectedErrors: []string{"node ID cannot contain any of the following characters:"},
		},
		{
			name: "Missing Orchestrators in Non-Requester Node",
			config: NATSTransportConfig{
				NodeID:     "node2",
				AuthSecret: "sekret",
			},
			expectedErrors: []string{"missing orchestrators"},
		},
		{
			name: "Missing Port in Requester Node",
			config: NATSTransportConfig{
				NodeID:          "node3",
				IsRequesterNode: true,
				AuthSecret:      "sekret",
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
				AuthSecret:               "sekret",
			},
			expectedErrors: []string{"cluster port -1 must be greater than zero"},
		},
		{
			name: "ServerTLSCert is set but ServerTLSKey not set",
			config: NATSTransportConfig{
				NodeID:        "nodeID",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
				Port:          1234,
				ServerTLSCert: "path/to/cert",
			},
			expectedErrors: []string{"both ServerTLSCert and ServerTLSKey must be set together"},
		},
		{
			name: "ServerTLSKey is set but ServerTLSCert not set",
			config: NATSTransportConfig{
				NodeID:        "nodeID",
				Orchestrators: []string{"orch1", "orch2"},
				AuthSecret:    "sekret",
				Port:          1234,
				ServerTLSKey:  "path/to/key",
			},
			expectedErrors: []string{"both ServerTLSCert and ServerTLSKey must be set together"},
		},
		{
			name: "TLSTimeout cannot be negative",
			config: NATSTransportConfig{
				NodeID:           "nodeID",
				Orchestrators:    []string{"orch1", "orch2"},
				AuthSecret:       "sekret",
				Port:             1234,
				ServerTLSKey:     "path/to/key",
				ServerTLSCert:    "path/to/cert",
				ServerTLSTimeout: -1,
			},
			expectedErrors: []string{"NATS ServerTLSTimeout must be a positive number, got: -1"},
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
