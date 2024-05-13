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
	suite.NotNil(txCtx.cancelFunc)

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

// Test suite runner.
func TestTxContextTestSuite(t *testing.T) {
	suite.Run(t, new(TxContextTestSuite))
}
