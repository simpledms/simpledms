package previewconversion

import "testing"

func TestClassifyEligibleSources(t *testing.T) {
	tests := []struct {
		name       string
		mimeType   string
		filename   string
		wantFamily Family
		wantInput  string
	}{
		{name: "html extension", mimeType: "application/octet-stream", filename: "Report.HTML", wantFamily: FamilyHTML, wantInput: "index.html"},
		{name: "markdown mime", mimeType: "text/markdown; charset=utf-8", filename: "README", wantFamily: FamilyMarkdown, wantInput: "source.md"},
		{name: "office extension", mimeType: "application/octet-stream", filename: "budget.XLSX", wantFamily: FamilyOffice, wantInput: "budget.XLSX"},
		{name: "office mime", mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", filename: "document", wantFamily: FamilyOffice, wantInput: "document.docx"},
		{name: "mime wins unknown extension", mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", filename: "document.bin", wantFamily: FamilyOffice, wantInput: "document.docx"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classification, eligible := Classify(test.mimeType, test.filename, false)
			if !eligible {
				t.Fatal("source was not classified as eligible")
			}
			if classification.Family != test.wantFamily {
				t.Fatalf("family = %q, want %q", classification.Family, test.wantFamily)
			}
			if classification.InputFilename != test.wantInput {
				t.Fatalf("input filename = %q, want %q", classification.InputFilename, test.wantInput)
			}
		})
	}
}

func TestClassifyExcludedSources(t *testing.T) {
	for _, test := range []struct {
		mimeType    string
		filename    string
		isDirectory bool
	}{
		{mimeType: "text/plain", filename: "notes.txt"},
		{mimeType: "application/pdf", filename: "already.pdf"},
		{mimeType: "image/png", filename: "image.png"},
		{mimeType: "application/zip", filename: "archive.zip"},
		{mimeType: "application/zip", filename: "spreadsheet.ods"},
		{mimeType: "application/x-zip-compressed", filename: "spreadsheet.ods"},
		{mimeType: "image/png", filename: "document.docx"},
		{mimeType: "text/html", filename: "folder", isDirectory: true},
	} {
		if classification, eligible := Classify(test.mimeType, test.filename, test.isDirectory); eligible || classification != nil {
			t.Fatalf("%q should not be eligible: %#v", test.filename, classification)
		}
	}
}

func TestPreviewFilename(t *testing.T) {
	if got, want := PreviewFilename("Invoice.DOCX"), "Invoice.pdf"; got != want {
		t.Fatalf("PreviewFilename() = %q, want %q", got, want)
	}
	if got, want := PreviewFilename("README"), "README.pdf"; got != want {
		t.Fatalf("PreviewFilename() = %q, want %q", got, want)
	}
}
