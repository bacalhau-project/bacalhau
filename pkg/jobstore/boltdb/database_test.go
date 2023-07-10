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

	dir, _ := os.MkdirTemp("", "bacalhau-jobstore")
	s.dbFile = filepath.Join(dir, "testing.db")

	s.store, _ = NewBoltJobStore(s.dbFile)
}

func (s *DatabaseTestSuite) TearDownTest() {
	//s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *DatabaseTestSuite) TestBucketCreation() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		final, err := GetBucketByPath(tx, "root.bucket.final", true)
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

func (s *DatabaseTestSuite) TestBucketCreationOne() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		final, err := GetBucketByPath(tx, "root", true)
		s.NoError(err)
		s.NotNil(final)
		return nil
	})
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestBucketCreationError() {
	err := s.store.database.Update(func(tx *bolt.Tx) error {
		_ = tx.Bucket([]byte("root"))

		final, err := GetBucketByPath(tx, "root.missing.bucket", false)
		s.Error(err)
		s.Nil(final)
		return nil
	})
	s.NoError(err)
}
