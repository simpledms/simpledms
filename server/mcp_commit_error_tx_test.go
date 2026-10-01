package server

import (
	"errors"

	"entgo.io/ent/dialect"
)

type mcpCommitErrorTx struct {
	dialect.Tx
	driver *mcpCommitErrorDriver
}

func (tx *mcpCommitErrorTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	if tx.driver.armed.CompareAndSwap(true, false) {
		tx.driver.injected.Store(true)
		return errors.New("injected post-commit error")
	}
	return nil
}
