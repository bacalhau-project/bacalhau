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

type DatabaseTestSuite struct {
	suite.Suite
	store  *BoltJobStore
	dbFile string
	ctx    context.Context
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

func (s *DatabaseTestSuite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-jobstore-test")
	s.dbFile = filepath.Join(dir, "testing.db")

	s.store, _ = NewBoltJobStore(s.dbFile)
}

func (s *DatabaseTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *DatabaseTestSuite) TestGetDatabaseBad() {
	_, err := GetDatabase("")
	s.Error(err)
}

func (s *DatabaseTestSuite) TestGetBucketDataErr() {
	_ = s.store.database.View(func(tx *bolt.Tx) (err error) {
		data := GetBucketData(tx, NewBucketPath("non-existent"), []byte("nope"))
		s.Nil(data)
		return nil
	})
}

func (s *DatabaseTestSuite) TestBucketCreation() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		final, err := NewBucketPath("root/bucket/final").Get(tx, true)
		s.NoError(err)
		s.NotNil(final)

		root := tx.Bucket([]byte("root"))
		s.NotNil(root)

		bucket := root.Bucket([]byte("bucket"))
		s.NotNil(bucket)

		finalAgain := bucket.Bucket([]byte("final"))
		s.NotNil(finalAgain)

		s.Equal(final.Root(), finalAgain.Root())

		return nil
	})
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestBucketPartialSearch() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_, err := NewBucketPath("root/notbucket-000").Get(tx, true)
		s.NoError(err)

		_, err = NewBucketPath("root/bucket-123").Get(tx, true)
		s.NoError(err)

		_, err = NewBucketPath("root/bucket-456").Get(tx, true)
		s.NoError(err)

		root := tx.Bucket([]byte("root"))
		s.NotNil(root)

		keys, err := GetBucketsByPrefix(tx, root, []byte("bucket"))
		s.NoError(err)

		s.Equal(2, len(keys))
		s.Equal("bucket-123", string(keys[0]))
		s.Equal("bucket-456", string(keys[1]))

		return nil
	})
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestBucketCreationOne() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		final, err := NewBucketPath("root").Get(tx, true)
		s.NoError(err)
		s.NotNil(final)
		return nil
	})
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestBucketCreationNone() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("single"))

		final, err := NewBucketPath("single").Get(tx, false)
		s.NoError(err)
		s.NotNil(final)

		return nil
	})
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestBucketCreationError() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_ = tx.Bucket([]byte("root"))

		final, err := NewBucketPath("root/missing/bucket").Get(tx, false)
		s.Error(err)
		s.Nil(final)
		return nil
	})
	s.NoError(err)
}
