package devstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ShardingSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestShardingSuite(t *testing.T) {
	suite.Run(t, new(ShardingSuite))
}

// Before all suite
func (suite *ShardingSuite) SetupAllSuite() {

}

// Before each test
func (suite *ShardingSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *ShardingSuite) TearDownTest() {
}

func (suite *ShardingSuite) TearDownAllSuite() {

}

func (suite *ShardingSuite) TestEndToEnd() {

	const nodeCount = 3
	// ctx, span := newSpan("sharding_endtoend")
	// defer span.End()

	stack, cm := SetupTest(
		suite.T(),
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	dirPath, err := os.MkdirTemp("", "sharding-test")
	require.NoError(suite.T(), err)

	for i := 0; i < 100; i++ {
		err = os.WriteFile(
			fmt.Sprintf("%s/%d.txt", dirPath, i),
			[]byte(fmt.Sprintf("hello %d", i)),
			0644,
		)
		require.NoError(suite.T(), err)
	}

	directoryCid, err := stack.AddFileToNodes(nodeCount, dirPath)

	fmt.Printf("directoryCid --------------------------------------\n")
	spew.Dump(directoryCid)
}
