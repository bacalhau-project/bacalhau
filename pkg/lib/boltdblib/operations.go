package boltdblib

import (
	"context"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// Update is a helper function that will update the job in the store.
// It checks context cancellation before starting any operation.
func Update(ctx context.Context, db *bolt.DB, update func(tx *bolt.Tx) error) error {
	// Check context cancellation before starting
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before starting update: %w", err)
	}

	// Check for existing transaction in context
	tx, externalTx := TxFromContext(ctx)
	if externalTx {
		if !tx.Writable() {
			return fmt.Errorf("readonly transaction provided in context for update operation")
		}
		return update(tx)
	}

	// Start new writable transaction
	tx, err := db.Begin(true)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always rollback on error for internally created transactions
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = update(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// View is a helper function that will perform a read-only operation on the store.
// It checks context cancellation before starting any operation.
func View(ctx context.Context, db *bolt.DB, view func(tx *bolt.Tx) error) error {
	// Check context cancellation before starting
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before starting view: %w", err)
	}

	// Check for existing transaction in context
	tx, externalTx := TxFromContext(ctx)
	if externalTx {
		return view(tx)
	}

	// Start new read-only transaction
	tx, err := db.Begin(false)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Always rollback read-only transactions
	defer tx.Rollback() // nolint: errcheck

	return view(tx)
}
