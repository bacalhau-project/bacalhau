//go:build unit || !integration

package boltjobstore

import (
	"context"
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

// Test_newTxContext tests if the transaction context is initialized correctly.
func (suite *TxContextTestSuite) Test_newTxContext() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := newTxContext(context.Background(), tx)
	suite.NotNil(txCtx.tx)

	// Ensure the transaction is part of the context.
	retrievedTx, ok := txFromContext(txCtx)
	suite.True(ok)
	suite.Equal(tx, retrievedTx)

	// Rollback to release the transaction.
	suite.NoError(txCtx.Rollback())
}

// Test_Commit tests committing a transaction through txContext.
func (suite *TxContextTestSuite) Test_Commit() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := newTxContext(context.Background(), tx)
	err = txCtx.Commit()
	suite.NoError(err)
	suite.Nil(txCtx.tx.DB()) // Ensure the transaction is committed.

	// Check if the transaction is committed by attempting another transaction.
	err = suite.db.View(func(tx *bolt.Tx) error {
		return nil // If no error, commit was successful.
	})
	suite.NoError(err)
}

// Test_Rollback tests rolling back a transaction through txContext.
func (suite *TxContextTestSuite) Test_Rollback() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := newTxContext(context.Background(), tx)
	err = txCtx.Rollback()
	suite.NoError(err)
	suite.Nil(txCtx.tx.DB()) // Ensure the transaction is rolled back.
}

// Test_AutomaticRollback tests automatic rollback on context cancellation.
func (suite *TxContextTestSuite) Test_AutomaticRollback() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := newTxContext(ctx, tx)

	cancel() // Trigger the cancellation.
	// Ensure the transaction is rolled back.
	suite.Eventually(func() bool {
		return txCtx.tx.DB() == nil
	}, 500*time.Millisecond, 10*time.Millisecond)

}

// Test_CommitAfterRollbackFails tests that commit after rollback fails.
func (suite *TxContextTestSuite) Test_CommitAfterRollbackFails() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := newTxContext(context.Background(), tx)
	suite.NoError(txCtx.Rollback())
	suite.Error(txCtx.Commit(), "expected error when committing after rollback")
}

// Test_CommitAfterCancelFails tests that commit after context cancellation fails.
func (suite *TxContextTestSuite) Test_CommitAfterCancelFails() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := newTxContext(ctx, tx)

	cancel() // Trigger the cancellation.
	suite.Eventually(func() bool { return txCtx.tx.DB() == nil }, 500*time.Millisecond, 10*time.Millisecond)
	suite.Error(txCtx.Commit(), "expected error when committing after context cancellation")
}

// Test_RollbackAfterCommitIsGraceful tests that rollback after commit is graceful.
func (suite *TxContextTestSuite) Test_RollbackAfterCommitIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := newTxContext(context.Background(), tx)
	suite.NoError(txCtx.Commit())
	suite.NoError(txCtx.Rollback(), "expected no error when rolling back after commit")
}

// Test_RollbackMultipleTimesIsGraceful tests that rollback multiple times is graceful.
func (suite *TxContextTestSuite) Test_RollbackMultipleTimesIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	txCtx := newTxContext(context.Background(), tx)
	suite.NoError(txCtx.Rollback())

	// Try to rollback multiple times.
	suite.NoError(txCtx.Rollback())
	suite.NoError(txCtx.Rollback())
}

// Test_RollbackAfterCancelIsGraceful tests that rollback after context cancellation is graceful.
func (suite *TxContextTestSuite) Test_RollbackAfterCancelIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := newTxContext(ctx, tx)

	cancel() // Trigger the cancellation.
	suite.Eventually(func() bool { return txCtx.tx.DB() == nil }, 500*time.Millisecond, 10*time.Millisecond)
	suite.NoError(txCtx.Rollback(), "expected no error when rolling back after context cancellation")
}

func (suite *TxContextTestSuite) Test_CancelAfterCommitIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := newTxContext(ctx, tx)
	suite.NoError(txCtx.Commit())
	cancel()                          // Trigger the cancellation.
	time.Sleep(50 * time.Millisecond) // Wait for the goroutine to handle the cancellation.
}

func (suite *TxContextTestSuite) Test_CancelAfterRollbackIsGraceful() {
	tx, err := suite.db.Begin(true)
	suite.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	txCtx := newTxContext(ctx, tx)
	suite.NoError(txCtx.Rollback())
	cancel()                          // Trigger the cancellation.
	time.Sleep(50 * time.Millisecond) // Wait for the goroutine to handle the cancellation.
}

// Test suite runner.
func TestTxContextTestSuite(t *testing.T) {
	suite.Run(t, new(TxContextTestSuite))
}
