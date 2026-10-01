package file

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/e"
)

type FileReader struct{}

func NewFileReader() *FileReader {
	return &FileReader{}
}

func (qq *FileReader) Get(ctx ctxx.Context, publicID string) (*enttenant.File, error) {
	file, err := ctx.SpaceCtx().Space.QueryFiles().Where(
		dbfile.PublicID(entx.NewCIText(publicID)),
	).Only(ctx)
	if err != nil {
		log.Printf("read scoped file: %T", err)
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusNotFound, "File not found.")
		}
		return nil, err
	}
	return file, nil
}

// TextWindow uses character offsets without copying the whole OCR string into a rune slice.
func (qq *FileReader) TextWindow(text string, offset, length int) (string, bool, error) {
	if offset < 0 || length < 1 || length > 50000 {
		return "", false, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid text range.")
	}
	start, end, count := len(text), len(text), 0
	for position := range text {
		if count == offset {
			start = position
		}
		if count-offset == length {
			end = position
			return text[start:end], true, nil
		}
		count++
	}
	return text[start:end], false, nil
}
