package server

import (
	"bytes"
	"database/sql"
	"errors"
	"log"
	"net/http"
)

// transactionResponse holds a small command response until persistence succeeds.
type transactionResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newTransactionResponse() *transactionResponse {
	return &transactionResponse{header: make(http.Header)}
}

func (qq *transactionResponse) Header() http.Header {
	return qq.header
}

func (qq *transactionResponse) WriteHeader(status int) {
	if qq.status == 0 {
		qq.status = status
	}
}

func (qq *transactionResponse) Write(data []byte) (int, error) {
	qq.WriteHeader(http.StatusOK)
	return qq.body.Write(data)
}

func (qq *transactionResponse) flush(rw http.ResponseWriter) {
	for key, values := range qq.header {
		rw.Header()[key] = values
	}
	if qq.status != 0 {
		rw.WriteHeader(qq.status)
	}
	if _, err := qq.body.WriteTo(rw); err != nil {
		log.Println(err)
	}
}

func rollbackResponseTransaction(tx interface{ Rollback() error }) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		log.Println(err)
	}
}
