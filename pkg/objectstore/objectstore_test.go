//go:build unit || !integration

package objectstore_test

import (
	"context"
	"fmt"
	"strings"
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

	type testdata struct {
		Name string
	}

	data := testdata{Name: "Test"}

	err = impl.Get(s.ctx, "test", "irrelevant", &data)
	require.Error(s.T(), err)
	require.EqualError(s.T(), err, "unknown prefix: test")

	err = impl.Get(s.ctx, "job", "nosuchkey", &data)
	require.Error(s.T(), err)
	require.EqualError(s.T(), err, "unknown key: nosuchkey")

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalBatchGet() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	type testdata struct {
		ID string
	}

	data1 := testdata{ID: "1"}
	data2 := testdata{ID: "2"}
	data3 := testdata{ID: "3"}

	_ = impl.Put(s.ctx, "job", data1.ID, data1)
	_ = impl.Put(s.ctx, "job", data2.ID, data2)
	_ = impl.Put(s.ctx, "job", data3.ID, data3)

	var results []testdata

	err = impl.GetBatch(s.ctx, "job", []string{"1", "2", "3"}, &results)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "1", results[0].ID)
	require.Equal(s.T(), 3, len(results))

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalBatchGetSingle() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	type testdata struct {
		ID string
	}

	data1 := testdata{ID: "1"}

	_ = impl.Put(s.ctx, "job", data1.ID, data1)

	var results []testdata

	err = impl.GetBatch(s.ctx, "job", []string{"1"}, &results)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "1", results[0].ID)

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalBatchGetNone() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	type testdata struct {
		ID string
	}

	var results []testdata
	err = impl.GetBatch(s.ctx, "job", []string{"1"}, &results)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []testdata{}, results)

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalReadAndWrite() {
	impl, err := objectstore.GetImplementation(objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	type testdata struct {
		Data string
	}

	data := testdata{Data: "some data"}

	err = impl.Put(s.ctx, "job", "key", data)
	require.NoError(s.T(), err)

	toFill := testdata{}
	err = impl.Get(s.ctx, "job", "key", &toFill)
	require.NoError(s.T(), err)
	require.Equal(s.T(), data, toFill)

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

	t := testdata{}
	err = impl.Get(s.ctx, "job", "key", &t)
	require.NoError(s.T(), err)
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

	impl.CallbackHooks().RegisterUpdate("job", userCallback)

	err = impl.Put(s.ctx, "job", "key", data)
	require.NoError(s.T(), err)

	// We now expect tags/tagname to contain a list of "1"
	var tagList []string
	err = impl.Get(s.ctx, "tags", "tagname", &tagList)
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

	impl.CallbackHooks().RegisterUpdate("job", userCallback)

	err = impl.Put(s.ctx, "job", data1.ID, data1)
	require.NoError(s.T(), err)

	err = impl.Put(s.ctx, "job", data3.ID, data3)
	require.NoError(s.T(), err)

	err = impl.Put(s.ctx, "job", data2.ID, data2)
	require.NoError(s.T(), err)

	// We now expect tags/tagname to contain a list of "1"
	var tagList []string
	err = impl.Get(s.ctx, "tags", "tagname", &tagList)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []string{"1", "2", "3"}, tagList)

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestLocalDelete() {
	type testdata struct {
		ID   string
		Name string
	}

	data1 := testdata{ID: "1", Name: "test"}
	userUpdateCallback := func(object any) ([]commands.Command, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := commands.NewCommand("tags", "tagname", commands.AddToSet(t.ID))
		return []commands.Command{c}, nil
	}

	userDeleteCallback := func(object any) ([]commands.Command, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := commands.NewCommand("tags", "tagname", commands.DeleteFromSet(t.ID))
		return []commands.Command{c}, nil
	}

	impl, err := objectstore.GetImplementation(
		objectstore.LocalImplementation,
		local.WithPrefixes("job", "tags"),
	)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	impl.CallbackHooks().RegisterUpdate("job", userUpdateCallback)
	impl.CallbackHooks().RegisterDelete("job", userDeleteCallback)

	err = impl.Put(s.ctx, "job", data1.ID, data1)
	require.NoError(s.T(), err)

	retr := testdata{}
	err = impl.Get(s.ctx, "job", data1.ID, retr)
	require.NoError(s.T(), err)

	err = impl.Delete(s.ctx, "job", data1.ID, data1)
	require.NoError(s.T(), err)

	// The tag name should now be an empty list
	var tagList []string
	err = impl.Get(s.ctx, "tags", "tagname", tagList)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []string(nil), tagList)
}

func (s *ObjectStoreTestSuite) TestLocalMapCallbacks() {
	type testdata struct {
		ID     string
		Name   string
		Labels map[string]string
	}

	data1 := testdata{
		ID:   "1",
		Name: "test",
		Labels: map[string]string{
			"height": "tall",
			"depth":  "deep",
		},
	}

	data2 := testdata{
		ID:   "2",
		Name: "another test",
		Labels: map[string]string{
			"height": "tall",
			"depth":  "shallow",
		},
	}

	userUpdateCallback := func(object any) ([]commands.Command, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		var commandList []commands.Command

		// If labels is
		//    Height=1
		//    Depth=2
		// we will end up with prefixes
		// /labels/height -> {"1": [t.ID]}
		// /labels/depth -> {"1": [t.ID]}
		for k, v := range t.Labels {
			// TODO: the ToLower should be slugify
			c := commands.NewCommand("labels", strings.ToLower(k), commands.AddToMap(v, t.ID))
			commandList = append(commandList, c)
		}

		return commandList, nil
	}

	userDeleteCallback := func(object any) ([]commands.Command, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		var commandList []commands.Command

		for k, v := range t.Labels {
			c := commands.NewCommand("labels", strings.ToLower(k), commands.DeleteFromMap(v, t.ID))
			commandList = append(commandList, c)
		}

		return commandList, nil
	}

	impl, err := objectstore.GetImplementation(
		objectstore.LocalImplementation,
		local.WithPrefixes("job", "labels"),
	)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	impl.CallbackHooks().RegisterUpdate("job", userUpdateCallback)
	impl.CallbackHooks().RegisterDelete("job", userDeleteCallback)

	err = impl.Put(s.ctx, "job", data1.ID, data1)
	require.NoError(s.T(), err)

	err = impl.Put(s.ctx, "job", data2.ID, data2)
	require.NoError(s.T(), err)

	checkLabels := func(name string, key string, values []string) {
		var labelMap map[string][]string

		err = impl.Get(s.ctx, "labels", name, &labelMap)
		require.NoError(s.T(), err)
		require.Equal(s.T(), values, labelMap[key])
	}

	checkLabels("height", "tall", []string{"1", "2"})

	err = impl.Delete(s.ctx, "job", data1.ID, data1)
	require.NoError(s.T(), err)

	checkLabels("height", "tall", []string{"2"})

	err = impl.Delete(s.ctx, "job", data2.ID, data2)
	require.NoError(s.T(), err)

	checkLabels("height", "tall", []string{})

}
