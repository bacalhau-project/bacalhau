//go:build unit || !integration

package objectstore_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/distributed"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/index"
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

func (s *ObjectStoreTestSuite) makeLocal(prefixes ...string) objectstore.ObjectStore {
	impl, err := objectstore.GetImplementation(s.ctx, objectstore.LocalImplementation, local.WithPrefixes(prefixes...))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)
	return impl
}

func (s *ObjectStoreTestSuite) makeDistributed() objectstore.ObjectStore {
	impl, err := objectstore.GetImplementation(s.ctx, objectstore.DistributedImplementation, distributed.WithTestConfig())
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)
	return impl
}

func (s *ObjectStoreTestSuite) TestCreateLocal() {
	impl, err := objectstore.GetImplementation(s.ctx, objectstore.LocalImplementation)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)
	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestCreateDistributed() {
	impl, err := objectstore.GetImplementation(
		s.ctx, objectstore.DistributedImplementation, distributed.WithTestConfig(),
	)
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)
	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestCreateLocalBadOption() {
	opt := distributed.WithPeers([]string{""})

	impl, err := objectstore.GetImplementation(
		s.ctx,
		objectstore.LocalImplementation,
		opt,
	)
	require.Nil(s.T(), impl)
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.ErrInvalidOption)
}

func (s *ObjectStoreTestSuite) TestCreateDistributedBadOption() {
	opt := local.WithDataFile("")

	impl, err := objectstore.GetImplementation(
		s.ctx,
		objectstore.DistributedImplementation,
		opt,
	)
	require.Nil(s.T(), impl)
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.ErrInvalidOption)
}

func (s *ObjectStoreTestSuite) TestLocalPrefixes() {
	impl, err := objectstore.GetImplementation(s.ctx, objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	type testdata struct {
		Name string
	}

	data := testdata{Name: "Test"}

	found, err := impl.Get(s.ctx, "test", "irrelevant", &data)
	require.Error(s.T(), err)
	require.False(s.T(), found)
	require.EqualError(s.T(), err, "unknown prefix: test")

	found, err = impl.Get(s.ctx, "job", "nosuchkey", &data)
	require.Error(s.T(), err)
	require.False(s.T(), found)
	require.EqualError(s.T(), err, "unknown key: nosuchkey")

	impl.Close(s.ctx)
}

func (s *ObjectStoreTestSuite) TestBatchGet() {
	impl, err := objectstore.GetImplementation(s.ctx, objectstore.LocalImplementation, local.WithPrefixes("job"))
	require.NotNil(s.T(), impl)
	require.NoError(s.T(), err)

	type testdata struct {
		ID string
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data1 := testdata{ID: "1"}
		data2 := testdata{ID: "2"}
		data3 := testdata{ID: "3"}

		_ = impl.Put(s.ctx, "job", data1.ID, data1)
		_ = impl.Put(s.ctx, "job", data2.ID, data2)
		_ = impl.Put(s.ctx, "job", data3.ID, data3)

		var results []testdata

		found, err := impl.GetBatch(s.ctx, "job", []string{"1", "2", "3"}, &results)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "1", results[0].ID)
		require.Equal(t, 3, len(results))

		impl.Close(s.ctx)
	}

	s.T().Run("Local Batch Get - Local", func(t *testing.T) {
		l := s.makeLocal("job")
		f(l, s.T())
	})
	s.T().Run("Local Batch Get - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestBatchGetSingle() {
	type testdata struct {
		ID string
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data1 := testdata{ID: "1"}
		_ = impl.Put(s.ctx, "job", data1.ID, data1)

		var results []testdata

		found, err := impl.GetBatch(s.ctx, "job", []string{"1"}, &results)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "1", results[0].ID)

		impl.Close(s.ctx)
	}

	s.T().Run("Local Batch Single - Local", func(t *testing.T) {
		l := s.makeLocal("job")
		f(l, s.T())
	})
	s.T().Run("Local Batch Single - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestBatchGetNone() {
	type testdata struct {
		ID string
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		var results []testdata
		found, err := impl.GetBatch(s.ctx, "job", []string{"1"}, &results)
		require.Error(t, err) // No such key
		require.False(t, found)
		require.Nil(t, results)
	}

	s.T().Run("Batch None - Local", func(t *testing.T) {
		l := s.makeLocal("job")
		f(l, s.T())
		l.Close(s.ctx)
	})
	s.T().Run("Batch None - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
		l.Close(s.ctx)
	})
}

func (s *ObjectStoreTestSuite) TestReadAndWrite() {
	type testdata struct {
		Data string
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data := testdata{Data: "some data"}

		err := impl.Put(s.ctx, "job", "key", data)
		require.NoError(s.T(), err)

		toFill := testdata{}
		found, err := impl.Get(s.ctx, "job", "key", &toFill)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, data, toFill)

		impl.Close(s.ctx)
	}

	s.T().Run("Read and Write - Local", func(t *testing.T) {
		l := s.makeLocal("job")
		f(l, s.T())
	})
	s.T().Run("Read and Write - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestReadAndWriteObject() {
	type testdata struct {
		Name string
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data := testdata{Name: "test"}

		err := impl.Put(s.ctx, "job", "key", data)
		require.NoError(t, err)

		test := testdata{}
		found, err := impl.Get(s.ctx, "job", "key", &test)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "test", test.Name)

		impl.Close(s.ctx)
	}
	s.T().Run("Read and Write Object - Local", func(t *testing.T) {
		l := s.makeLocal("job")
		f(l, s.T())
	})
	s.T().Run("Read and Write Object - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestReadAndWriteObjectWithCallbacks() {

	type testdata struct {
		ID   string
		Name string
	}

	userCallback := func(object any) ([]index.IndexCommand, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := index.NewIndexCommand("tags", "tagname", index.AddToSet(t.ID))
		return []index.IndexCommand{c}, nil
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data := testdata{ID: "1", Name: "test"}

		impl.CallbackHooks().RegisterUpdate("job", userCallback)

		err := impl.Put(s.ctx, "job", "key", data)
		require.NoError(t, err)

		// We now expect tags/tagname to contain a list of "1"
		var tagList []string
		found, err := impl.Get(s.ctx, "tags", "tagname", &tagList)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, data.ID, tagList[0])

		impl.Close(s.ctx)
	}

	s.T().Run("Read and Write Callbacks - Local", func(t *testing.T) {
		l := s.makeLocal("job", "tags")
		f(l, s.T())
	})
	s.T().Run("Read and Write Callbacks - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestLocalReadAndWriteObjectWithMultipleCallbacks() {
	type testdata struct {
		ID   string
		Name string
	}

	userCallback := func(object any) ([]index.IndexCommand, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := index.NewIndexCommand("tags", "tagname", index.AddToSet(t.ID))
		return []index.IndexCommand{c}, nil
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data1 := testdata{ID: "1", Name: "test"}
		data2 := testdata{ID: "2", Name: "test"}
		data3 := testdata{ID: "3", Name: "test"}

		impl.CallbackHooks().RegisterUpdate("job", userCallback)

		err := impl.Put(s.ctx, "job", data1.ID, data1)
		require.NoError(t, err)

		err = impl.Put(s.ctx, "job", data3.ID, data3)
		require.NoError(t, err)

		err = impl.Put(s.ctx, "job", data2.ID, data2)
		require.NoError(t, err)

		// We now expect tags/tagname to contain a list of "1"
		var tagList []string
		found, err := impl.Get(s.ctx, "tags", "tagname", &tagList)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"1", "2", "3"}, tagList)

		impl.Close(s.ctx)
	}

	s.T().Run("Read and Write Multiple Callbacks - Local", func(t *testing.T) {
		l := s.makeLocal("job", "tags")
		f(l, s.T())
	})
	s.T().Run("Read and Write Callbacks - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestDelete() {
	type testdata struct {
		ID   string
		Name string
	}

	userUpdateCallback := func(object any) ([]index.IndexCommand, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := index.NewIndexCommand("tags", "tagname", index.AddToSet(t.ID))
		return []index.IndexCommand{c}, nil
	}

	userDeleteCallback := func(object any) ([]index.IndexCommand, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		c := index.NewIndexCommand("tags", "tagname", index.DeleteFromSet(t.ID))
		return []index.IndexCommand{c}, nil
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
		data1 := testdata{ID: "1", Name: "test"}

		impl.CallbackHooks().RegisterUpdate("job", userUpdateCallback)
		impl.CallbackHooks().RegisterDelete("job", userDeleteCallback)

		err := impl.Put(s.ctx, "job", data1.ID, data1)
		require.NoError(t, err)

		retr := testdata{}
		found, err := impl.Get(s.ctx, "job", data1.ID, retr)
		require.NoError(t, err)
		require.True(t, found)

		err = impl.Delete(s.ctx, "job", data1.ID, data1)
		require.NoError(t, err)

		// The tag name should now be an empty list
		var tagList []string
		found, err = impl.Get(s.ctx, "tags", "tagname", tagList)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string(nil), tagList)

		impl.Close(s.ctx)
	}

	s.T().Run("Delete - Local", func(t *testing.T) {
		l := s.makeLocal("job", "tags")
		f(l, s.T())
	})
	s.T().Run("Delete - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}

func (s *ObjectStoreTestSuite) TestMapCallbacks() {
	type testdata struct {
		ID     string
		Name   string
		Labels map[string]string
	}

	userUpdateCallback := func(object any) ([]index.IndexCommand, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		var commandList []index.IndexCommand

		// If labels is
		//    Height=1
		//    Depth=2
		// we will end up with prefixes
		// /labels/height -> {"1": [t.ID]}
		// /labels/depth -> {"1": [t.ID]}
		for k, v := range t.Labels {
			// TODO: the ToLower should be slugify
			c := index.NewIndexCommand("labels", strings.ToLower(k), index.AddToMap(v, t.ID))
			commandList = append(commandList, c)
		}

		return commandList, nil
	}

	userDeleteCallback := func(object any) ([]index.IndexCommand, error) {
		t, ok := object.(testdata)
		if !ok {
			return nil, fmt.Errorf("callback type did not match: got %T", object)
		}

		var commandList []index.IndexCommand

		for k, v := range t.Labels {
			c := index.NewIndexCommand("labels", strings.ToLower(k), index.DeleteFromMap(v, t.ID))
			commandList = append(commandList, c)
		}

		return commandList, nil
	}

	f := func(impl objectstore.ObjectStore, t *testing.T) {
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

		impl.CallbackHooks().RegisterUpdate("job", userUpdateCallback)
		impl.CallbackHooks().RegisterDelete("job", userDeleteCallback)

		err := impl.Put(s.ctx, "job", data1.ID, data1)
		require.NoError(t, err)

		err = impl.Put(s.ctx, "job", data2.ID, data2)
		require.NoError(t, err)

		checkLabels := func(name string, key string, values []string) {
			var labelMap map[string][]string

			found, err := impl.Get(s.ctx, "labels", name, &labelMap)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, values, labelMap[key])
		}

		checkLabels("height", "tall", []string{"1", "2"})

		err = impl.Delete(s.ctx, "job", data1.ID, data1)
		require.NoError(t, err)

		checkLabels("height", "tall", []string{"2"})

		err = impl.Delete(s.ctx, "job", data2.ID, data2)
		require.NoError(t, err)

		checkLabels("height", "tall", []string{})
		impl.Close(s.ctx)
	}

	s.T().Run("Map Callbacks - Local", func(t *testing.T) {
		l := s.makeLocal("job", "labels")
		f(l, s.T())
	})
	s.T().Run("Map - Distributed", func(t *testing.T) {
		l := s.makeDistributed()
		f(l, s.T())
	})
}
