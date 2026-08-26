package filesource

import "testing"

func TestFileSourceStringsMatchPersistedValues(t *testing.T) {
	tests := map[FileSource]string{
		UnknownLegacy:    "UnknownLegacy",
		WebInterface:     "WebInterface",
		PWAOSOpen:        "PWAOSOpen",
		URLImport:        "URLImport",
		WebDAV:           "WebDAV",
		SystemExtraction: "SystemExtraction",
	}

	for source, want := range tests {
		if got := source.String(); got != want {
			t.Fatalf("String() = %q, want %q", got, want)
		}
	}
}
