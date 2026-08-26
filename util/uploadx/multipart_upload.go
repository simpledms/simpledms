package uploadx

import (
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/simpledms/simpledms/util/e"
)

const multipartOverheadBytes int64 = 1 * 1024 * 1024

// MultipartFile carries the streaming file part and optional declared size.
type MultipartFile struct {
	Reader        io.Reader
	Closer        io.Closer
	Filename      string
	ExpectedBytes *int64
}

// LimitMultipartBody applies the configured upload limit plus multipart framing headroom.
func LimitMultipartBody(rw http.ResponseWriter, req *http.Request, nilableUploadLimitBytes *int64) {
	if nilableUploadLimitBytes == nil {
		return
	}
	bodyLimitBytes := *nilableUploadLimitBytes
	if bodyLimitBytes < math.MaxInt64-multipartOverheadBytes {
		bodyLimitBytes += multipartOverheadBytes
	}
	req.Body = http.MaxBytesReader(rw, req.Body, bodyLimitBytes)
}

// NewMultipartFile wraps a multipart file part and validates its optional Content-Length.
func NewMultipartFile(part *multipart.Part) (*MultipartFile, error) {
	file := &MultipartFile{
		Reader:   part,
		Closer:   part,
		Filename: part.FileName(),
	}
	if rawSize := part.Header.Get("Content-Length"); rawSize != "" {
		size, err := strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid upload size.")
		}
		file.ExpectedBytes = &size
	}
	return file, nil
}
