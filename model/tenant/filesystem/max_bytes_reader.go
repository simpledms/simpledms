package filesystem

import "io"

type maxBytesReader struct {
	r    io.Reader
	max  int64
	read int64
}

func (qq *maxBytesReader) Read(p []byte) (int, error) {
	if qq.max >= 0 && qq.read >= qq.max {
		var extra [1]byte
		n, err := qq.r.Read(extra[:])
		qq.read += int64(n)
		if n > 0 {
			return 0, errUploadTooLarge
		}
		return 0, err
	}
	if qq.max >= 0 {
		remaining := qq.max - qq.read
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
	}
	n, err := qq.r.Read(p)
	qq.read += int64(n)
	return n, err
}
