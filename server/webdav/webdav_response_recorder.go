package webdav

import (
	"bytes"
	"net/http"
	"strings"
)

type webDAVResponseRecorder struct {
	rw     http.ResponseWriter
	head   http.Header
	status int
	body   bytes.Buffer
}

func newWebDAVResponseRecorder(rw http.ResponseWriter) *webDAVResponseRecorder {
	return &webDAVResponseRecorder{
		rw:   rw,
		head: make(http.Header),
	}
}

func (qq *webDAVResponseRecorder) Header() http.Header { return qq.head }

func (qq *webDAVResponseRecorder) WriteHeader(status int) {
	if qq.status == 0 {
		qq.status = status
	}
}

func (qq *webDAVResponseRecorder) Write(data []byte) (int, error) {
	if qq.status == 0 {
		qq.status = http.StatusOK
	}
	if qq.body.Len()+len(data) <= webDAVMaxStatusBody {
		_, _ = qq.body.Write(data)
	}
	return len(data), nil
}

func (qq *webDAVResponseRecorder) flush(op *webDAVOperation) {
	status := qq.status
	if status == 0 {
		status = http.StatusOK
	}
	if op.err != nil {
		if mapped, ok := webDAVMappedHTTPStatus(op.err); ok {
			status = mapped
		}
	}
	for key, values := range qq.head {
		if strings.EqualFold(key, "ETag") && op.method == "PUT" {
			continue
		}
		for _, value := range values {
			qq.rw.Header().Add(key, value)
		}
	}
	if status == http.StatusMethodNotAllowed {
		qq.rw.Header().Set("Allow", webDAVAllow)
	}
	qq.rw.WriteHeader(status)
	if status != http.StatusNoContent {
		body := qq.body.Bytes()
		if op.err != nil && status >= 400 {
			body = []byte(http.StatusText(status))
		}
		_, _ = qq.rw.Write(body)
	}
}
