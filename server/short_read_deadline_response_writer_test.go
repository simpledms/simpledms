package server

import (
	"net/http"
	"time"
)

type shortReadDeadlineResponseWriter struct {
	http.ResponseWriter
}

func (qq *shortReadDeadlineResponseWriter) SetReadDeadline(deadline time.Time) error {
	// Exercise the real transport deadline without waiting for the production timeout.
	if !deadline.IsZero() {
		deadline = time.Now().Add(50 * time.Millisecond)
	}
	return http.NewResponseController(qq.ResponseWriter).SetReadDeadline(deadline)
}

func (qq *shortReadDeadlineResponseWriter) Unwrap() http.ResponseWriter {
	return qq.ResponseWriter
}
