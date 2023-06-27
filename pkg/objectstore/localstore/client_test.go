//go:build unit || !integration

package localstore_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/localstore"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LocalClientTestSuite struct {
	suite.Suite
	ctx   context.Context
	store *localstore.LocalStore
}

func TestLocalClientTestSuite(t *testing.T) {
	suite.Run(t, new(LocalClientTestSuite))
}

func (s *LocalClientTestSuite) SetupTest() {
	s.store, _ = localstore.NewLocalStore(localstore.WithTestLocation(), localstore.WithPrefixes("tests", "containers"))
}

func (s *LocalClientTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
}

type TestData struct {
	ContainerID string
	ID          string
	Name        string
	Age         int
}

func (t TestData) OnUpdate() []objectstore.Indexer {
	return []objectstore.Indexer{
		objectstore.NewIndexer("containers", t.ContainerID, objectstore.AddToSetOperation(t.ID)),
	}
}

func (t TestData) OnDelete() []objectstore.Indexer {
	return []objectstore.Indexer{
		objectstore.NewIndexer("containers", t.ContainerID, objectstore.DeleteFromSetOperation(t.ID)),
	}
}

func (s *LocalClientTestSuite) TestSimpleMissingKey() {
	c := localstore.NewClient[TestData](s.ctx, "tests", s.store)
	_, err := c.Get("missing")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.NewErrNotFound("missing"))
}

func (s *LocalClientTestSuite) TestSimplePut() {
	t := TestData{Name: "Bob", Age: 30}

	c := localstore.NewClient[TestData](s.ctx, "tests", s.store)
	err := c.Put("bob", t)
	require.NoError(s.T(), err)
}

func (s *LocalClientTestSuite) TestPutGetDelete() {
	t := TestData{ID: "1", ContainerID: "100", Name: "Bob", Age: 30}

	c := localstore.NewClient[TestData](s.ctx, "tests", s.store)
	err := c.Put("bob", t)
	require.NoError(s.T(), err)

	t, err = c.Get("bob")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), t)
	require.Equal(s.T(), "Bob", t.Name)

	// containers should contain the a key of 100 with ID of 1
	data, err := c.GetStore().Get(s.ctx, "containers", "100")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "[\"1\"]", string(data))

	err = c.Delete("bob", t)
	require.NoError(s.T(), err)

	t, err = c.Get("bob")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.NewErrNotFound("bob"))

	// containers should NOT contain the a key of 100 with ID of 1
	var index []string
	bytes, err := c.GetStore().Get(s.ctx, "containers", "100")
	require.NoError(s.T(), err)

	err = json.Unmarshal(bytes, &index)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0, len(index))

}
