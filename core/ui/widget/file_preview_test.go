package widget

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestFilePreviewUsesFilenameWhenMimeTypeIsMissing(t *testing.T) {
	preview := &FilePreview{
		Filename: "notes.txt",
	}

	if !preview.IsPreviewable() {
		t.Fatal("expected .txt file with missing MIME type to be previewable")
	}
	if got, want := preview.GetMimeType(), "text/plain; charset=utf-8"; got != want {
		t.Fatalf("GetMimeType() = %q, want %q", got, want)
	}

	tmpl := template.Must(template.ParseFiles("file_preview.gohtml"))
	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "FilePreview", preview); err != nil {
		t.Fatalf("render FilePreview: %v", err)
	}
	rendered := output.String()
	if !strings.Contains(rendered, `<object`) ||
		!strings.Contains(rendered, `type="text/plain; charset=utf-8"`) {
		t.Fatalf("expected text preview object, got: %s", rendered)
	}
}
