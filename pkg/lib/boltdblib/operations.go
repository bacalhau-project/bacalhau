package boltdblib

import (
	"context"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// Update is a helper function that will update the job in the store
// it accepts a context, an update function and creates a new transaction to
// perform the update if no transaction is provided in the context
func Update(ctx context.Context, db *bolt.DB, update func(tx *bolt.Tx) error) error {
	var err error
	var tx *bolt.Tx

	// if ctx has a transaction value, then we can use that transaction, otherwise we need to create one
	var externalTx bool
	tx, externalTx = TxFromContext(ctx)
	if externalTx {
		if !tx.Writable() {
			return fmt.Errorf("readonly transaction provided in context for update operation")
		}
	} else {
		tx, err = db.Begin(true)
		if err != nil {
			return err
		}
	}

	// always rollback the transaction if there was an error
	// and the transaction was created internally in this call
	defer func() {
		if !externalTx && err != nil {
			_ = tx.Rollback()
		}
	}()

	err = update(tx)
	if err != nil {
		return err
	}

	// only commit the transaction if it was created internally in this call
	if !externalTx {
		err = tx.Commit()
	}
	return err
}

// View is a helper function that will perform a read-only operation on the store
// it accepts a context, a view function and creates a new transaction to
// perform the view if no transaction is provided in the context
func View(ctx context.Context, db *bolt.DB, view func(tx *bolt.Tx) error) error {
	var err error

	// if ctx has a transaction value, then we can use that transaction, otherwise we need to create one
	tx, externalTx := TxFromContext(ctx)
	if !externalTx {
		tx, err = db.Begin(false)
		if err != nil {
			return err
		}
	}

	// always rollback the transaction if the transaction
	// was created internally in this call
	// note that we don't commit the transaction as it's read-only
	defer func() {
		if !externalTx {
			_ = tx.Rollback()
		}
	}()

	return view(tx)
}
