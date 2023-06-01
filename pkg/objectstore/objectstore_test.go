//go:build unit || !integration

package objectstore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/commands"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/local"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ObjectStoreTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestObjectStoreTestSuite(t *testing.T) {
	suite.Run(t, &ObjectStoreTestSuite{
		ctx: context.Background(),
	})
}

func (s *ObjectStoreTestSuite) TestCreateLocal() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)
	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestCreateDistributed() {
	impl, err := objectstore.GetImplementation(objectstore.DistributedImplementation)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)
	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestCreateLocalBadOption() {
	opt := distributed.WithPeers([]string{""})

	impl, err := objectstore.GetImplementation(
		objectstore.LocalImplementation,
		opt,
	)
	require.Nil(s.T(), impl)
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.ErrInvalidOption)
}

func (s *ObjectStoreTestSuite) TestCreateDistributedBadOption() {
	opt := local.WithDataFolder("")

	impl, err := objectstore.GetImplementation(
		objectstore.DistributedImplementation,
		opt,
	)
	require.Nil(s.T(), impl)
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.ErrInvalidOption)
}

func (s *ObjectStoreTestSuite) TestLocalPrefixes() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	val, err := impl.Get(s.ctx, "test", "irrelevant")
	require.Error(s.T(), err)
	require.EqualError(s.T(), err, "unknown prefix: test")
	require.Nil(s.T(), val)

	val, err = impl.Get(s.ctx, "job", "nosuchkey")
	require.NoError(s.T(), err)
	require.Nil(s.T(), val)

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalReadAndWrite() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	data := []byte("some data")

	err = impl.Put(s.ctx, "job", "key", data)
	require.NoError(s.T(), err)

	value, err := impl.Get(s.ctx, "job", "key")
	require.NoError(s.T(), err)
	require.Equal(s.T(), data, value)

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalReadAndWriteObject() {

	type testdata struct {
		Name string
	}

	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	data := testdata{Name: "test"}

	err = impl.Put(s.ctx, "job", "key", data)
	require.NoError(s.T(), err)

	value, err := impl.Get(s.ctx, "job", "key")
	require.NoError(s.T(), err)

	var t testdata
	json.Unmarshal(value, &t)
	require.Equal(s.T(), "test", t.Name)

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalReadAndWriteObjectWithCallbacks() {

	type testdata struct {
		ID   string
		Name string
	}

	userCallback := func(object any) ([]commands.Command, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := commands.NewCommand("tags", "tagname", commands.AddToSet(t.ID))
		return []commands.Command{c}, nil
	}

	data := testdata{ID: "1", Name: "test"}

	impl, err := objectstore.GetImplementation(
		objectstore.LocalImplementation,
		local.WithPrefixes("job", "tags"),
	)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	impl.CallbackHooks().RegisterUpdate(testdata{}, userCallback)

	err = impl.Put(s.ctx, "job", "key", data)
	require.NoError(s.T(), err)

	// We now expect tags/tagname to contain a list of "1"
	tagListBytes, err := impl.Get(s.ctx, "tags", "tagname")
	require.NoError(s.T(), err)

	var tagList []string

	err = json.Unmarshal(tagListBytes, &tagList)
	require.NoError(s.T(), err)
	require.Equal(s.T(), data.ID, tagList[0])

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalReadAndWriteObjectWithMultipleCallbacks() {
	type testdata struct {
		ID   string
		Name string
	}

	data1 := testdata{ID: "1", Name: "test"}
	data2 := testdata{ID: "2", Name: "test"}
	data3 := testdata{ID: "3", Name: "test"}

	userCallback := func(object any) ([]commands.Command, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := commands.NewCommand("tags", "tagname", commands.AddToSet(t.ID))
		return []commands.Command{c}, nil
	}

	impl, err := objectstore.GetImplementation(
		objectstore.LocalImplementation,
		local.WithPrefixes("job", "tags"),
	)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	impl.CallbackHooks().RegisterUpdate(testdata{}, userCallback)

	err = impl.Put(s.ctx, "job", data1.ID, data1)
	require.NoError(s.T(), err)

	err = impl.Put(s.ctx, "job", data3.ID, data3)
	require.NoError(s.T(), err)

	err = impl.Put(s.ctx, "job", data2.ID, data2)
	require.NoError(s.T(), err)

	// We now expect tags/tagname to contain a list of "1"
	tagListBytes, err := impl.Get(s.ctx, "tags", "tagname")
	require.NoError(s.T(), err)

	var tagList []string

	err = json.Unmarshal(tagListBytes, &tagList)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []string{"1", "2", "3"}, tagList)

	impl.Close(s.ctx)
}
