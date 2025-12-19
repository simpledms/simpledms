package sqlx

import (
	"context"
	"errors"
	"log"
)

type client[T, U any] interface {
	Debug() T
	Tx(ctx context.Context) (U, error)
	Close() error
}

type DB[T client[T, U], U any] struct {
	ReadOnlyConn           T
	ReadWriteConn          T
	readWriteDataSourceURL string
}

func newDB[T client[T, U], U any](readOnlyConn, readWriteConn T, readWriteDataSourceURL string) *DB[T, U] {
	return &DB[T, U]{
		ReadOnlyConn:           readOnlyConn,
		ReadWriteConn:          readWriteConn,
		readWriteDataSourceURL: readWriteDataSourceURL,
	}
}

func (qq *DB[T, U]) ReadWriteDataSourceURL() string {
	return qq.readWriteDataSourceURL
}

func (qq *DB[T, U]) Debug() {
	qq.ReadWriteConn = qq.ReadWriteConn.Debug()
	qq.ReadOnlyConn = qq.ReadOnlyConn.Debug()
}

func (qq *DB[T, U]) Tx(ctx context.Context, isReadOnly bool) (U, error) {
	if isReadOnly {
		return qq.ReadOnlyConn.Tx(ctx)
	}
	return qq.ReadWriteConn.Tx(ctx)
}

func (qq *DB[T, U]) Close() error {
	err := qq.ReadWriteConn.Close()
	if err != nil {
		log.Println(err)
	}
	errx := qq.ReadOnlyConn.Close()
	if errx != nil {
		log.Println(errx)
	}
	return errors.Join(err, errx)
}
