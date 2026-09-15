package filesystem

import (
	"io"
	"strings"
	"testing"
)

func TestMimeSniffingReaderDetectsPlainText(t *testing.T) {
	reader := &mimeSniffingReader{r: strings.NewReader("plain text contents")}

	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("read contents: %v", err)
	}
	if got, want := reader.MimeType(), "text/plain; charset=utf-8"; got != want {
		t.Fatalf("MIME type = %q, want %q", got, want)
	}
}
