package server

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	maxPublicRequestBodyBytes = 1024 * 1024
	publicRequestBodyTimeout  = 20 * time.Second
)

func bufferPublicRequestBody(rw http.ResponseWriter, req *http.Request) bool {
	if req.Body == nil || req.Body == http.NoBody {
		return true
	}

	controller := http.NewResponseController(rw)
	err := controller.SetReadDeadline(time.Now().Add(publicRequestBodyTimeout))
	if err == nil {
		defer func() {
			if err := controller.SetReadDeadline(time.Time{}); err != nil {
				log.Println(err)
			}
		}()
	} else if !errors.Is(err, http.ErrNotSupported) {
		log.Println(err)
		rw.Header().Set("Connection", "close")
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return false
	}

	req.Body = http.MaxBytesReader(rw, req.Body, maxPublicRequestBodyBytes)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("public request body rejected: %T", err)
		status := http.StatusBadRequest
		var maxBytesErr *http.MaxBytesError
		var networkErr net.Error
		switch {
		case errors.As(err, &maxBytesErr):
			status = http.StatusRequestEntityTooLarge
		case errors.As(err, &networkErr) && networkErr.Timeout():
			status = http.StatusRequestTimeout
		}
		rw.Header().Set("Connection", "close")
		http.Error(rw, http.StatusText(status), status)
		return false
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	return true
}
