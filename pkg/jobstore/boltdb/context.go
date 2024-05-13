package boltjobstore

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	bolt "go.etcd.io/bbolt"
)

// contextKey is a custom type to avoid key collisions in context values.
type contextKey int

// txContextKey is the key used to store the transaction context in the context.
const txContextKey contextKey = 0

// txContext extends context.Context with transaction specific functionality.
type txContext struct {
	context.Context
	tx         *bolt.Tx
	cancelFunc context.CancelFunc
}

// newTxContext creates a new transactional context for a BoltDB transaction.
// It embeds a standard context and manages transaction commit/rollback based on the context's lifecycle.
func newTxContext(ctx context.Context, tx *bolt.Tx) *txContext {
	innerCtx, cancelFunc := context.WithCancel(context.WithValue(ctx, txContextKey, tx))
	txCtx := &txContext{
		Context:    innerCtx,
		tx:         tx,
		cancelFunc: cancelFunc,
	}
	// Start a goroutine that listens for the context's Done channel.
	go func() {
		<-innerCtx.Done()
		// If the transaction has not been committed when the context is done, rollback the transaction.
		// In boltdb transaction, db is nil if the transaction is already committed or rolled back.
		if txCtx.tx.DB() != nil {
			_ = txCtx.tx.Rollback()
		}
	}()

	return txCtx
}

// txFromContext retrieves the transaction from the context, if available.
func txFromContext(ctx context.Context) (*bolt.Tx, bool) {
	tx, ok := ctx.Value(txContextKey).(*bolt.Tx)
	return tx, ok
}

// Commit commits the transaction and cancels the context.
func (b *txContext) Commit() error {
	err := b.tx.Commit()
	b.cancelFunc()
	return err
}

// Rollback rolls back the transaction and cancels the context.
func (b *txContext) Rollback() error {
	err := b.tx.Rollback()
	b.cancelFunc()
	return err
}

// compile time check whether the txContext implements the TxContext interface from the jobstore package.
var _ jobstore.TxContext = &txContext{}
