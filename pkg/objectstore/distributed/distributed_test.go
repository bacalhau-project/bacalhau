//go:build unit || !integration

package distributed_test

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DistributedTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestDistributedTestSuite(t *testing.T) {
	suite.Run(t, &DistributedTestSuite{
		ctx: context.Background(),
	})
}

type testdata struct {
	Name string
}

func (s *DistributedTestSuite) TestCreate() {
	impl, err := objectstore.GetImplementation(
		s.ctx, objectstore.DistributedImplementation, distributed.WithTestConfig(),
	)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	t1 := &testdata{Name: "bob"}
	t2 := &testdata{}

	err = impl.Put(s.ctx, "p", "k", &t1)
	require.NoError(s.T(), err)

	found, err := impl.Get(s.ctx, "p", "k", &t2)
	require.NoError(s.T(), err)
	require.True(s.T(), found)
	require.Equal(s.T(), t1, t2)

	impl.Close(s.ctx)
}
