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

type BucketPathTestSuite struct {
	suite.Suite
	store  *BoltJobStore
	dbFile string
	ctx    context.Context
}

func TestBucketPathTestSuite(t *testing.T) {
	suite.Run(t, new(BucketPathTestSuite))
}

func (s *BucketPathTestSuite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-jobstore-test")
	s.dbFile = filepath.Join(dir, "testing.db")

	s.store, _ = NewBoltJobStore(s.dbFile)
}

func (s *BucketPathTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *BucketPathTestSuite) TestGetDatabaseBad() {
	_, err := GetDatabase("")
	s.Error(err)
}

func (s *BucketPathTestSuite) TestBucketCreation() {
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

func (s *BucketPathTestSuite) TestBucketCreationOne() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		final, err := NewBucketPath("root").Get(tx, true)
		s.NoError(err)
		s.NotNil(final)
		return nil
	})
	s.NoError(err)
}

func (s *BucketPathTestSuite) TestBucketCreationNone() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("single"))

		final, err := NewBucketPath("single").Get(tx, false)
		s.NoError(err)
		s.NotNil(final)

		return nil
	})
	s.NoError(err)
}

func (s *BucketPathTestSuite) TestBucketCreationError() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_ = tx.Bucket([]byte("root"))

		final, err := NewBucketPath("root/missing/bucket").Get(tx, false)
		s.Error(err)
		s.Nil(final)
		return nil
	})
	s.NoError(err)
}
