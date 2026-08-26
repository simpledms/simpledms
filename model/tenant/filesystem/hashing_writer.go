package filesystem

import (
	"hash"
	"io"
)

type hashingWriter struct {
	w      io.Writer
	sha256 hash.Hash
	crc32c hash.Hash32
	count  int64
}

func (qq *hashingWriter) Write(p []byte) (int, error) {
	n, err := qq.w.Write(p)
	if n > 0 {
		chunk := p[:n]
		qq.count += int64(n)
		_, _ = qq.sha256.Write(chunk)
		_, _ = qq.crc32c.Write(chunk)
	}
	return n, err
}
