//go:build unit || !integration

package boltjobstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

type IndexTestSuite struct {
	suite.Suite
	store  *BoltJobStore
	dbFile string
	ctx    context.Context
}

func TestIndexTestSuite(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}

func (s *IndexTestSuite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-jobstore-test")
	s.dbFile = filepath.Join(dir, "index-testing.db")

	s.store, _ = NewBoltJobStore(s.dbFile)
	s.store.database.Update(func(tx *bolt.Tx) (err error) {
		tx.CreateBucketIfNotExists([]byte("test"))
		return
	})
}

func (s *IndexTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *IndexTestSuite) TestIndexUsage() {
	err := s.store.database.Update(func(tx *bolt.Tx) (err error) {
		i := NewIndex("test/tags")

		label := []byte("hasgpu")

		err = i.Add(tx, []byte("94b136a3"), label)
		s.NoError(err)

		entries, err := i.List(tx, label)
		s.NoError(err)
		s.Equal(1, len(entries))
		s.Equal("94b136a3", string(entries[0]))

		err = i.Remove(tx, []byte("94b136a3"), label)
		s.NoError(err)

		entries, err = i.List(tx, label)
		s.NoError(err)
		s.Equal(0, len(entries))

		return nil
	})
	s.NoError(err)
}
