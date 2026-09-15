package mimetypex

import "testing"

func TestResolve(t *testing.T) {
	for _, test := range []struct {
		name     string
		mimeType string
		filename string
		want     string
	}{
		{
			name:     "stored MIME",
			mimeType: "text/plain; charset=iso-8859-1",
			filename: "notes.txt",
			want:     "text/plain; charset=iso-8859-1",
		},
		{
			name:     "text extension fallback",
			filename: "notes.TXT",
			want:     "text/plain; charset=utf-8",
		},
		{
			name:     "unknown extension",
			filename: "file.unknown-extension",
			want:     "application/octet-stream",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Resolve(test.mimeType, test.filename); got != test.want {
				t.Fatalf("Resolve() = %q, want %q", got, test.want)
			}
		})
	}
}
