package inbox

import (
	"testing"

	"github.com/simpledms/simpledms/util/ocrutil"
)

func TestOcrStatusMessageReturnsNilWhenOCRSucceeded(t *testing.T) {
	msg := ocrStatusMessage(true, 10)
	if msg != nil {
		t.Fatalf("expected nil message when OCR already succeeded")
	}
}

func TestOcrStatusMessageForPendingOCR(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeEnvVar, "100")

	msg := ocrStatusMessage(false, 100)
	if msg == nil {
		t.Fatalf("expected message for pending OCR")
	}

	want := "Text recognition (OCR) is not ready yet, suggestions are based on the filename only."
	if msg.StringUntranslated() != want {
		t.Fatalf("expected %q, got %q", want, msg.StringUntranslated())
	}
}

func TestOcrStatusMessageForTooLargeFile(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeEnvVar, "100")

	msg := ocrStatusMessage(false, 101)
	if msg == nil {
		t.Fatalf("expected message for too-large OCR file")
	}

	want := "Text recognition (OCR) cannot be applied because the file is too large, suggestions are based on the filename only."
	if msg.StringUntranslated() != want {
		t.Fatalf("expected %q, got %q", want, msg.StringUntranslated())
	}
}
