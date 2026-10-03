package server

import "net/http/httptest"

type commitObservationWriter struct {
	*httptest.ResponseRecorder
	observe          func() bool
	committedAtWrite bool
	done             bool
}

func (qq *commitObservationWriter) WriteHeader(status int) {
	if !qq.done {
		qq.done = true
		qq.committedAtWrite = qq.observe()
	}
	qq.ResponseRecorder.WriteHeader(status)
}

func (qq *commitObservationWriter) Write(body []byte) (int, error) {
	if !qq.done {
		qq.done = true
		qq.committedAtWrite = qq.observe()
	}
	return qq.ResponseRecorder.Write(body)
}
