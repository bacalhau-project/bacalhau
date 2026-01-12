package boltdblib

import (
	"context"
	"errors"

	bolt "go.etcd.io/bbolt"
)

// TxContext is a transactional context that can be used to commit or rollback
type TxContext interface {
	context.Context
	Commit() error
	Rollback() error
}

// contextKey is a custom type to avoid key collisions in context values
type contextKey int

// txContextKey is the key used to store the transaction context in the context
const txContextKey contextKey = 0

// txContext provides a simple wrapper around BoltDB transactions
type txContext struct {
	context.Context
	tx *bolt.Tx
}

// NewTxContext creates a new transaction context
func NewTxContext(ctx context.Context, tx *bolt.Tx) TxContext {
	return &txContext{
		Context: context.WithValue(ctx, txContextKey, tx),
		tx:      tx,
	}
}

// TxFromContext retrieves the transaction from the context, if available
func TxFromContext(ctx context.Context) (*bolt.Tx, bool) {
	tx, ok := ctx.Value(txContextKey).(*bolt.Tx)
	return tx, ok
}

// Commit commits the transaction
func (t *txContext) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *txContext) Rollback() error {
	if err := t.tx.Rollback(); err != nil && !errors.Is(err, bolt.ErrTxClosed) { //nolint:staticcheck // TODO: migrate to bbolt/errors package
		return err
	}
	return nil
}
