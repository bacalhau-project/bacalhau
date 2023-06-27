//go:build unit || !integration

package localstore_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/localstore"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LocalTestSuite struct {
	suite.Suite
	ctx   context.Context
	store *localstore.LocalStore
}

func TestLocalTestSuite(t *testing.T) {
	suite.Run(t, new(LocalTestSuite))
}

func (s *LocalTestSuite) SetupTest() {
	s.ctx = context.TODO()
	s.store, _ = localstore.NewLocalStore(localstore.WithTestLocation(), localstore.WithPrefixes("tests", "containers"))
}

func (s *LocalTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
}

func (s *LocalTestSuite) TestLocalPrefixes() {
	_, err := s.store.Get(s.ctx, "badprefix", "irrelevant")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.NewErrInvalidPrefix("badprefix"))

	_, err = s.store.Get(s.ctx, "tests", "nosuchkey")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, objectstore.NewErrNotFound("nosuchkey"))
}

func (s *LocalTestSuite) TestBatchGet() {
	type testdata struct {
		ID string
	}

	data1 := testdata{ID: "1"}
	data2 := testdata{ID: "2"}
	data3 := testdata{ID: "3"}

	data1B, _ := json.Marshal(data1)
	data2B, _ := json.Marshal(data2)
	data3B, _ := json.Marshal(data3)

	_ = s.store.Put(s.ctx, "tests", data1.ID, data1B)
	_ = s.store.Put(s.ctx, "tests", data2.ID, data2B)
	_ = s.store.Put(s.ctx, "tests", data3.ID, data3B)

	m, err := s.store.GetBatch(s.ctx, "tests", []string{"1", "2", "3"})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 3, len(m))

	m, err = s.store.GetBatch(s.ctx, "tests", []string{"1", "5", "4"})
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, len(m))
}

func (s *LocalTestSuite) TestBatchGetNone() {
	results, err := s.store.GetBatch(s.ctx, "tests", []string{"1"})
	require.NoError(s.T(), err)

	t, ok := results["1"]
	require.False(s.T(), ok)
	require.Nil(s.T(), t)
}

func (s *LocalTestSuite) TestReadAndWrite() {
	type testdata struct {
		Data string
	}

	data := testdata{Data: "some data"}
	dataB, _ := json.Marshal(&data)

	err := s.store.Put(s.ctx, "tests", "key", dataB)
	require.NoError(s.T(), err)

	b, err := s.store.Get(s.ctx, "tests", "key")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), b)

	data = testdata{}
	err = json.Unmarshal(b, &data)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "some data", data.Data)
}

func (s *LocalTestSuite) TestDelete() {
	type testdata struct {
		ID   string
		Name string
	}

	data1 := testdata{ID: "1", Name: "test"}
	data1B, _ := json.Marshal(data1)

	err := s.store.Put(s.ctx, "tests", data1.ID, data1B)
	require.NoError(s.T(), err)

	retr := testdata{}
	retrB, err := s.store.Get(s.ctx, "tests", data1.ID)
	require.NoError(s.T(), err)

	err = json.Unmarshal(retrB, &retr)
	require.NoError(s.T(), err)

	err = s.store.Delete(s.ctx, "tests", data1.ID)
	require.NoError(s.T(), err)
}

func (s *LocalTestSuite) TestList() {
	type testdata struct {
		Key string
	}

	data := []testdata{
		{Key: "Jobs1"},
		{Key: "Jobs2"},
		{Key: "Jobs3"},
	}
	for _, tc := range data {
		b, _ := json.Marshal(tc)
		_ = s.store.Put(s.ctx, "tests", tc.Key, b)
	}

	keys, _ := s.store.List(s.ctx, "tests", "Jobs")
	for _, tc := range data {
		require.Contains(s.T(), keys, tc.Key)
	}

}

func BenchmarkWrite(b *testing.B) {
	ctx := context.Background()
	store, _ := localstore.NewLocalStore(localstore.WithTestLocation(), localstore.WithPrefixes("test"))

	bytes := []byte(`{"name": "bob", "age": 100}`)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = store.Put(ctx, "test", strconv.Itoa(n), bytes)
	}
	store.Close(ctx)

	b.ReportAllocs()
}

func BenchmarkRead(b *testing.B) {
	ctx := context.Background()
	store, _ := localstore.NewLocalStore(localstore.WithTestLocation(), localstore.WithPrefixes("test"))

	bytes := []byte(`{"name": "bob", "age": 100}`)
	for n := 0; n < 100; n++ {
		_ = store.Put(ctx, "test", strconv.Itoa(n), bytes)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = store.Get(ctx, "test", "50")
	}

	store.Close(ctx)

	b.ReportAllocs()
}
