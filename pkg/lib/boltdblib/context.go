package boltdblib

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
)

// contextKey is a custom type to avoid key collisions in context values.
type contextKey int

// txContextKey is the key used to store the transaction context in the context.
const txContextKey contextKey = 0

// TxContext extends context.Context with transaction specific functionality.
// Note:
// boltdb transactions are not thread-safe, and we have to synchronize access to the transaction
// while trying to rollback the transaction on context cancellation.
// This might add some overhead, and it might make sense to delegate the handling of context cancellation
// to the caller, but this is a trade-off to ensure that the transaction is always rolled back on context cancellation.
// TODO: Evaluate the trade-offs and consider delegating the handling of context cancellation to the caller.
type TxContext struct {
	context.Context
	tx      *bolt.Tx
	closed  bool
	closeCh chan struct{}
	mu      sync.Mutex
}

// NewTxContext creates a new transactional context for a BoltDB transaction.
// It embeds a standard context and manages transaction commit/rollback based on the context's lifecycle.
func NewTxContext(ctx context.Context, tx *bolt.Tx) *TxContext {
	innerCtx := context.WithValue(ctx, txContextKey, tx)
	txCtx := &TxContext{
		Context: innerCtx,
		tx:      tx,
		closeCh: make(chan struct{}),
	}
	// Start a goroutine that listens for the context's Done channel.
	go func() {
		defer func() {
			// Attempt to rollback the transaction, which is a no-op if already committed or rolled back.
			if err := txCtx.doRollback(); err != nil {
				log.Ctx(txCtx).Error().Err(err).Msg("failed to rollback transaction on tx cleanup")
			}
		}()
		select {
		case <-innerCtx.Done():
		case <-txCtx.closeCh:
		}
	}()

	return txCtx
}

// TxFromContext retrieves the transaction from the context, if available.
func TxFromContext(ctx context.Context) (*bolt.Tx, bool) {
	tx, ok := ctx.Value(txContextKey).(*bolt.Tx)
	return tx, ok
}

// Commit commits the transaction and cancels the context.
// Commit will return an error if the transaction is already committed or rolled back.
func (b *TxContext) Commit() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer b.close()
	return b.tx.Commit()
}

// Rollback rolls back the transaction and cancels the context.
// Rollback is a no-op if the transaction is already committed or rolled back.
func (b *TxContext) Rollback() error {
	return b.doRollback()
}

// doRollback is a helper function to rollback the transaction without cancelling the context.
func (b *TxContext) doRollback() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer b.close()
	if err := b.tx.Rollback(); err != nil && !errors.Is(err, bolt.ErrTxClosed) {
		return err
	}
	return nil
}

// close closes the transactional context.
// already called with the mutex held.
func (b *TxContext) close() {
	if !b.closed {
		close(b.closeCh)
		b.closed = true
	}
}

// compile time check whether the TxContext implements the TxContext interface from the jobstore package.
var _ jobstore.TxContext = &TxContext{}
