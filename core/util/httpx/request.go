package httpx

import (
	"net/http"
)

type Request struct {
	*http.Request
}

func NewRequest(request *http.Request) *Request {
	return &Request{Request: request}
}
