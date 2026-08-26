package filesystem

import (
	"io"
	"net/http"
)

type mimeSniffingReader struct {
	r   io.Reader
	buf []byte
}

func (qq *mimeSniffingReader) Read(p []byte) (int, error) {
	n, err := qq.r.Read(p)
	if n > 0 && len(qq.buf) < 512 {
		remaining := 512 - len(qq.buf)
		if n < remaining {
			remaining = n
		}
		qq.buf = append(qq.buf, p[:remaining]...)
	}
	return n, err
}

func (qq *mimeSniffingReader) MimeType() string {
	if len(qq.buf) == 0 {
		return ""
	}
	return http.DetectContentType(qq.buf)
}
