package filesystem

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestDetectMIMEForZIPBasedDocuments(t *testing.T) {
	tests := []struct {
		name     string
		entries  map[string]string
		wantMIME string
	}{
		{
			name: "ODS",
			entries: map[string]string{
				"mimetype": "application/vnd.oasis.opendocument.spreadsheet",
			},
			wantMIME: "application/vnd.oasis.opendocument.spreadsheet",
		},
		{
			name: "DOCX",
			entries: map[string]string{
				"[Content_Types].xml": "",
				"word/document.xml":   "",
			},
			wantMIME: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:     "plain ZIP",
			entries:  map[string]string{"document.txt": "content"},
			wantMIME: "application/zip",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var content bytes.Buffer
			archive := zip.NewWriter(&content)
			for name, value := range test.entries {
				entry, err := archive.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := entry.Write([]byte(value)); err != nil {
					t.Fatal(err)
				}
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}

			mimeType, err := detectMIME(bytes.NewReader(content.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if mimeType != test.wantMIME {
				t.Fatalf("MIME type = %q, want %q", mimeType, test.wantMIME)
			}
		})
	}
}
