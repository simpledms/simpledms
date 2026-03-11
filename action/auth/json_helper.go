package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func writeJSONResponse(rw httpx.ResponseWriter, statusCode int, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Pragma", "no-cache")
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteData(data, statusCode)
	return nil
}

func decodeJSONBody(req *httpx.Request, target any) error {
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(target); err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid request payload.")
	}
	return nil
}
