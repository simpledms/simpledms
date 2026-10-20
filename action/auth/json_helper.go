package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func writeJSONResponse(rw httpx.ResponseWriter, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Pragma", "no-cache")
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteData(data, http.StatusOK)
	return nil
}

func decodeJSONBody(req *httpx.Request, target any) error {
	req.Body = http.MaxBytesReader(nil, req.Body, 1024*1024)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(target)
	if err == nil {
		var extra any
		err = decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
	if err != nil {
		log.Printf("JSON payload rejected: %T", err)
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Request payload is too large.")
		}
	}
	return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid request payload.")
}
