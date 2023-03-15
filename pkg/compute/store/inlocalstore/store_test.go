package inlocalstore

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

func TestZeroExecutionsReturnsZeroCount(t *testing.T) {
	ctx := context.Background()
	system.InitConfigForTesting(t)
	//use inmemorystore
	proxy := NewStoreProxy(inmemory.NewStore())
	count, err := proxy.GetExecutionCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint(0), count)

}

//func TestOneExecutionReturnsOneCount(t *testing.T) {
//	ctx := context.Background()
//	system.InitConfigForTesting(t)
//	proxy := NewStoreProxy(inmemory.NewStore())
//	err := proxy.CreateExecution(ctx, )
//	//create execution
//	//proxy.GetExecutionCount returns 1?

//Check that it reads the counter from the file without changing the file
//Check it gives the right value ALWAYS
//case : no executions - 0
// case : run execution and it's completed - 1
// case : run more than 1
// case: Check that ONLY completed states are counted
//create a proxy, give it nonzero count (i.e. a completed execution),
//create a NEW proxy and check that the value is the same.
