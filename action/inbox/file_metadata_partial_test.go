package inbox

import (
	"testing"

	"github.com/simpledms/simpledms/util/ocrutil"
)

func TestOcrStatusMessageReturnsNilWhenOCRSucceeded(t *testing.T) {
	partial := &FileMetadataPartial{}
	msg := partial.nilableOCRStatusMessage(true, 10)
	if msg != nil {
		t.Fatalf("expected nil message when OCR already succeeded")
	}
}

func TestOcrStatusMessageForPendingOCR(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	partial := &FileMetadataPartial{}
	msg := partial.nilableOCRStatusMessage(false, 100)
	if msg == nil {
		t.Fatalf("expected message for pending OCR")
	}

	want := "Text recognition (OCR) is not ready yet, suggestions are based on the filename only."
	if msg.StringUntranslated() != want {
		t.Fatalf("expected %q, got %q", want, msg.StringUntranslated())
	}
}

func TestOcrStatusMessageForTooLargeFile(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	partial := &FileMetadataPartial{}
	msg := partial.nilableOCRStatusMessage(false, (1024*1024)+1)
	if msg == nil {
		t.Fatalf("expected message for too-large OCR file")
	}

	want := "Text recognition (OCR) cannot be applied because the file is too large, suggestions are based on the filename only."
	if msg.StringUntranslated() != want {
		t.Fatalf("expected %q, got %q", want, msg.StringUntranslated())
	}
}
