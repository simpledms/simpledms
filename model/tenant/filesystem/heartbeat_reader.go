package filesystem

import (
	"io"
	"time"
)

type heartbeatReader struct {
	r          io.Reader
	onProgress func(time.Time)
	next       time.Time
}

func (qq *heartbeatReader) Read(p []byte) (int, error) {
	n, err := qq.r.Read(p)
	if n > 0 && qq.onProgress != nil {
		now := time.Now()
		if qq.next.IsZero() || !now.Before(qq.next) {
			qq.onProgress(now)
			qq.next = now.Add(uploadProgressInterval)
		}
	}
	return n, err
}
