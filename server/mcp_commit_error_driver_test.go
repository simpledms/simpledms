package server

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"

	"entgo.io/ent/dialect"
)

type mcpCommitErrorDriver struct {
	dialect.Driver
	armed    atomic.Bool
	injected atomic.Bool
}

func (d *mcpCommitErrorDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := d.Driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	return &mcpCommitErrorTx{Tx: tx, driver: d}, nil
}

func (d *mcpCommitErrorDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	beginner, ok := d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	})
	if !ok {
		return nil, errors.New("underlying tenant driver does not support BeginTx")
	}
	tx, err := beginner.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &mcpCommitErrorTx{Tx: tx, driver: d}, nil
}
