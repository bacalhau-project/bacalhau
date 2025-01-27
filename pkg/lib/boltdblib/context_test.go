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

type TxContextTestSuite struct {
	suite.Suite
	db *bolt.DB
}

func (suite *TxContextTestSuite) SetupTest() {
	// Open the Bolt database on the temporary file.
	db, err := bolt.Open(filepath.Join(suite.T().TempDir(), "context.db"), 0600, nil)
	suite.Require().NoError(err)
	suite.db = db
}

// TearDownSuite cleans up resources after all tests have run.
func (suite *TxContextTestSuite) TearDownTest() {
	_ = suite.db.Close()
}

func (suite *TxContextTestSuite) verifyTransactionClosed(tx *bolt.Tx) bool {
	return tx.DB() == nil
}

// Test_newTxContext tests if the transaction context is initialized correctly.
func (suite *TxContextTestSuite) Test_newTxContext() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)
	retrievedTx, ok := TxFromContext(txCtx)
	suite.True(ok)
	suite.Equal(tx, retrievedTx)

	// Rollback to clean up
	suite.Require().NoError(txCtx.Rollback())
}

// Test_Commit tests committing a transaction through TxContext.
func (suite *TxContextTestSuite) Test_Commit() {
	testBucket := []byte("test")

	// Create a transaction and try to create a bucket
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)

	_, err = tx.CreateBucket(testBucket)
	suite.Require().NoError(err)
	suite.Require().NoError(txCtx.Commit())
	suite.True(suite.verifyTransactionClosed(tx))

	// Verify the bucket was created (commit worked)
	err = suite.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(testBucket)
		if b == nil {
			return errors.New("bucket not found")
		}
		return nil
	})
	suite.Require().NoError(err, "bucket should exist after commit")
}

// Test_Rollback tests rolling back a transaction through TxContext.
func (suite *TxContextTestSuite) Test_Rollback() {
	testBucket := []byte("test")

	// Create a transaction and try to create a bucket
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)

	_, err = tx.CreateBucket(testBucket)
	suite.Require().NoError(err)
	suite.Require().NoError(txCtx.Rollback())
	suite.True(suite.verifyTransactionClosed(tx))

	// Verify the bucket was not created (rollback worked)
	err = suite.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(testBucket)
		if b != nil {
			return errors.New("bucket should not exist")
		}
		return nil
	})
	suite.Require().NoError(err, "bucket should not exist after rollback")
}

func (suite *TxContextTestSuite) Test_CommitAfterRollbackFails() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)
	suite.Require().NoError(txCtx.Rollback())
	suite.True(suite.verifyTransactionClosed(tx))
	suite.Error(txCtx.Commit(), "expected error when committing after rollback")
}

// Test_RollbackAfterCommitIsGraceful tests that rollback after commit is graceful.
func (suite *TxContextTestSuite) Test_RollbackAfterCommitIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)
	suite.Require().NoError(txCtx.Commit())
	suite.True(suite.verifyTransactionClosed(tx))
	suite.Require().NoError(txCtx.Rollback(), "expected no error when rolling back after commit")
}

// Test_RollbackMultipleTimesIsGraceful tests that rollback multiple times is graceful.
func (suite *TxContextTestSuite) Test_RollbackMultipleTimesIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)
	suite.Require().NoError(txCtx.Rollback())
	suite.True(suite.verifyTransactionClosed(tx))

	// Try to rollback multiple times.
	suite.Require().NoError(txCtx.Rollback())
	suite.Require().NoError(txCtx.Rollback())
}

// Test suite runner.
func TestTxContextTestSuite(t *testing.T) {
	suite.Run(t, new(TxContextTestSuite))
}
