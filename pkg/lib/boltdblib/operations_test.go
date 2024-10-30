//go:build unit || !integration

package boltdblib

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

type OperationsTestSuite struct {
	suite.Suite
	db *bolt.DB
}

func (suite *OperationsTestSuite) SetupTest() {
	db, err := bolt.Open(filepath.Join(suite.T().TempDir(), "operations.db"), 0600, nil)
	suite.Require().NoError(err)
	suite.db = db

	err = suite.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("testBucket"))
		return err
	})
	suite.Require().NoError(err)
}

func (suite *OperationsTestSuite) TearDownTest() {
	suite.NoError(suite.db.Close())
}

func (suite *OperationsTestSuite) TestUpdate() {
	// Test successful update
	err := Update(context.Background(), suite.db, func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("testBucket"))
		return b.Put([]byte("key"), []byte("value"))
	})
	suite.NoError(err)

	// Verify the update
	suite.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("testBucket"))
		v := b.Get([]byte("key"))
		suite.Equal([]byte("value"), v)
		return nil
	})

	// Test update with error
	err = Update(context.Background(), suite.db, func(tx *bolt.Tx) error {
		return errors.New("test error")
	})
	suite.Error(err)
	suite.Equal("test error", err.Error())

	// Test update with external transaction
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)
	defer tx.Rollback()

	ctx := NewTxContext(context.Background(), tx)
	err = Update(ctx, suite.db, func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("testBucket"))
		return b.Put([]byte("key2"), []byte("value2"))
	})
	suite.NoError(err)

	// Verify the update within the transaction
	v := tx.Bucket([]byte("testBucket")).Get([]byte("key2"))
	suite.Equal([]byte("value2"), v)

	// Test update with read-only external transaction
	txReadOnly, err := suite.db.Begin(false)
	suite.Require().NoError(err)
	defer txReadOnly.Rollback()

	ctxReadOnly := NewTxContext(context.Background(), txReadOnly)
	err = Update(ctxReadOnly, suite.db, func(tx *bolt.Tx) error {
		return nil
	})
	suite.Error(err)
	suite.Contains(err.Error(), "readonly transaction provided in context for update operation")
}

func (suite *OperationsTestSuite) TestView() {
	// Prepare some data
	suite.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("testBucket"))
		return b.Put([]byte("key"), []byte("value"))
	})

	// Test successful view
	err := View(context.Background(), suite.db, func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("testBucket"))
		v := b.Get([]byte("key"))
		suite.Equal([]byte("value"), v)
		return nil
	})
	suite.NoError(err)

	// Test view with error
	err = View(context.Background(), suite.db, func(tx *bolt.Tx) error {
		return errors.New("test error")
	})
	suite.Error(err)
	suite.Equal("test error", err.Error())

	// Test view with external transaction
	tx, err := suite.db.Begin(false)
	suite.Require().NoError(err)
	defer tx.Rollback()

	ctx := NewTxContext(context.Background(), tx)
	err = View(ctx, suite.db, func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("testBucket"))
		v := b.Get([]byte("key"))
		suite.Equal([]byte("value"), v)
		return nil
	})
	suite.NoError(err)
}

func TestOperationsTestSuite(t *testing.T) {
	suite.Run(t, new(OperationsTestSuite))
}
