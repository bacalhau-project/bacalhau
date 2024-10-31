//go:build unit || !integration

package boltdblib

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

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

// Test_AutomaticRollback tests automatic rollback on context cancellation.
func (suite *TxContextTestSuite) Test_AutomaticRollback() {
	testBucket := []byte("test")

	// Start a transaction that will be cancelled
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	NewTxContext(ctx, tx)

	_, err = tx.CreateBucket(testBucket)
	suite.Require().NoError(err)

	// Cancel the context and wait for rollback
	cancel()
	suite.Eventually(func() bool {
		return suite.verifyTransactionClosed(tx)
	}, 500*time.Millisecond, 10*time.Millisecond)

	// Verify the bucket was not created (automatic rollback worked)
	err = suite.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(testBucket)
		if b != nil {
			return errors.New("bucket should not exist")
		}
		return nil
	})
	suite.Require().NoError(err, "bucket should not exist after context cancellation")
}

// Test_CommitAfterRollbackFails tests that commit after rollback fails.
func (suite *TxContextTestSuite) Test_CommitAfterRollbackFails() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := NewTxContext(context.Background(), tx)
	suite.Require().NoError(txCtx.Rollback())
	suite.True(suite.verifyTransactionClosed(tx))
	suite.Error(txCtx.Commit(), "expected error when committing after rollback")
}

// Test_CommitAfterCancelFails tests that commit after context cancellation fails.
func (suite *TxContextTestSuite) Test_CommitAfterCancelFails() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := NewTxContext(ctx, tx)

	cancel() // Trigger the cancellation.
	suite.Eventually(func() bool {
		return suite.verifyTransactionClosed(tx)
	}, 500*time.Millisecond, 10*time.Millisecond)
	suite.Error(txCtx.Commit(), "expected error when committing after context cancellation")
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

// Test_RollbackAfterCancelIsGraceful tests that rollback after context cancellation is graceful.
func (suite *TxContextTestSuite) Test_RollbackAfterCancelIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := NewTxContext(ctx, tx)

	cancel() // Trigger the cancellation.
	suite.Eventually(func() bool {
		return suite.verifyTransactionClosed(tx)
	}, 500*time.Millisecond, 10*time.Millisecond)
	suite.Require().NoError(txCtx.Rollback(), "expected no error when rolling back after context cancellation")
}

func (suite *TxContextTestSuite) Test_CancelAfterCommitIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := NewTxContext(ctx, tx)
	suite.Require().NoError(txCtx.Commit())
	suite.True(suite.verifyTransactionClosed(tx))
	cancel()                          // Trigger the cancellation.
	time.Sleep(50 * time.Millisecond) // Wait for the goroutine to handle the cancellation.
}

func (suite *TxContextTestSuite) Test_CancelAfterRollbackIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := NewTxContext(ctx, tx)
	suite.Require().NoError(txCtx.Rollback())
	suite.True(suite.verifyTransactionClosed(tx))
	cancel()                          // Trigger the cancellation.
	time.Sleep(50 * time.Millisecond) // Wait for the goroutine to handle the cancellation.
}

// Test suite runner.
func TestTxContextTestSuite(t *testing.T) {
	suite.Run(t, new(TxContextTestSuite))
}
