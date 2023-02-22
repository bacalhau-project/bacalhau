//go:build unit || !integration

package publicapi

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	requester_publicapi "github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServerSuite struct {
	suite.Suite
	node   *node.Node
	client *requester_publicapi.RequesterAPIClient
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

// Before each test
func (s *ServerSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	n, client := setupNodeForTest(s.T())
	s.node = n
	s.client = client
}

// After each test
func (s *ServerSuite) TearDownTest() {
	s.node.CleanupManager.Cleanup(context.Background())
}

func (s *ServerSuite) TestList() {
	ctx := context.Background()

	// Should have no jobs initially:
	jobs, err := s.client.List(ctx, "", model.IncludeAny, model.ExcludeNone, 10, true, "created_at", true)
	require.NoError(s.T(), err)
	require.Empty(s.T(), jobs)

	// Submit a random job to the node:
	j := testutils.MakeNoopJob()

	_, err = s.client.Submit(ctx, j)
	require.NoError(s.T(), err)

	// Should now have one job:
	jobs, err = s.client.List(ctx, "", model.IncludeAny, model.ExcludeNone, 10, true, "created_at", true)
	require.NoError(s.T(), err)
	require.Len(s.T(), jobs, 1)
}

func (s *ServerSuite) TestSubmitRejectsJobWithSigilHeader() {
	j := testutils.MakeNoopJob()
	jobID, err := uuid.NewRandom()
	require.NoError(s.T(), err)

	s.client.DefaultHeaders["X-Bacalhau-Job-ID"] = jobID.String()
	_, err = s.client.Submit(context.Background(), j)
	require.Error(s.T(), err)
}
