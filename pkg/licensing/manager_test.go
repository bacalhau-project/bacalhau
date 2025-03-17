//go:build unit || !integration

package licensing

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
)

// StubReader is a simple implementation of the Reader interface
type StubReader struct {
	license *license.LicenseClaims
	token   string
}

func (r *StubReader) License() *license.LicenseClaims {
	return r.license
}

func (r *StubReader) RawToken() string {
	return r.token
}

// StubNodesTracker is a simple implementation of the nodes.Tracker interface
type StubNodesTracker struct {
	count int
}

func (t *StubNodesTracker) GetConnectedNodesCount() int {
	return t.count
}

type ManagerTestSuite struct {
	suite.Suite
	reader       *StubReader
	nodesTracker *StubNodesTracker
	manager      Manager
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.reader = &StubReader{}
	suite.nodesTracker = &StubNodesTracker{}

	params := ManagerParams{
		Reader:             suite.reader,
		NodesTracker:       suite.nodesTracker,
		ValidationInterval: 100 * time.Millisecond, // Short interval for testing
		SkipValidation:     false,
	}

	var err error
	suite.manager, err = NewManager(params)
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.manager)
}

func (suite *ManagerTestSuite) TestNewManager_InvalidParams() {
	// Test with nil reader
	params := ManagerParams{
		NodesTracker: suite.nodesTracker,
	}
	manager, err := NewManager(params)
	suite.Require().Error(err)
	suite.Require().Nil(manager)

	// Test with nil nodes tracker
	params = ManagerParams{
		Reader: suite.reader,
	}
	manager, err = NewManager(params)
	suite.Require().Error(err)
	suite.Require().Nil(manager)
}

func (suite *ManagerTestSuite) TestValidate_NoLicense() {
	suite.reader.license = nil
	suite.nodesTracker.count = 1

	state := suite.manager.Validate()
	suite.Require().Equal(LicenseValidationTypeFreeTierValid, state.Type)
	suite.Require().Equal(GetNoLicenseMessage(), state.Message)

	suite.nodesTracker.count = FreeTierMaxNodes + 1
	state = suite.manager.Validate()
	suite.Require().Equal(LicenseValidationTypeFreeTierExceeded, state.Type)
	suite.Require().Equal(GetFreeTierExceededMessage(FreeTierMaxNodes+1), state.Message)
}

func (suite *ManagerTestSuite) TestValidate_ExpiredLicense() {
	suite.reader.license = &license.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	suite.nodesTracker.count = 1

	state := suite.manager.Validate()
	suite.Require().Equal(LicenseValidationTypeExpired, state.Type)
	suite.Require().Equal(GetExpiredMessage(suite.reader.license.ExpiresAt.Time), state.Message)
}

func (suite *ManagerTestSuite) TestValidate_ExceededNodes() {
	suite.reader.license = &license.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Capabilities: map[string]string{
			"max_nodes": "2",
		},
	}
	suite.nodesTracker.count = 3

	state := suite.manager.Validate()
	suite.Require().Equal(LicenseValidationTypeExceededNodes, state.Type)
	suite.Require().Equal(GetExceededNodesMessage(3, 2), state.Message)
}

func (suite *ManagerTestSuite) TestValidate_ValidLicense() {
	suite.reader.license = &license.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		Capabilities: map[string]string{
			"max_nodes": "5",
		},
	}
	suite.nodesTracker.count = 3

	state := suite.manager.Validate()
	suite.Require().Equal(LicenseValidationTypeValid, state.Type)
	suite.Require().Equal(GetValidMessage(), state.Message)
}

func (suite *ManagerTestSuite) TestValidate_SkipValidation() {
	params := ManagerParams{
		Reader:             suite.reader,
		NodesTracker:       suite.nodesTracker,
		ValidationInterval: 100 * time.Millisecond,
		SkipValidation:     true,
	}

	manager, err := NewManager(params)
	suite.Require().NoError(err)
	suite.Require().NotNil(manager)

	state := manager.Validate()
	suite.Require().Equal(LicenseValidationTypeValid, state.Type)
	suite.Require().Equal(GetSkippedMessage(), state.Message)
}

func (suite *ManagerTestSuite) TestStartStop() {
	// Test stopping when not started
	suite.manager.Stop() // Should not panic

	// Test starting twice
	suite.manager.Start()
	suite.manager.Start() // Should not panic

	// Wait for a validation cycle
	time.Sleep(150 * time.Millisecond)

	// Test stopping twice
	suite.manager.Stop()
	suite.manager.Stop() // Should not panic

	// Test start, stop, stop sequence
	suite.manager.Start()
	time.Sleep(150 * time.Millisecond)
	suite.manager.Stop()
	suite.manager.Stop() // Should not panic
}

func (suite *ManagerTestSuite) TestLicense() {
	claims := &license.LicenseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	suite.reader.license = claims

	result := suite.manager.License()
	suite.Require().Equal(claims, result)
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
