package uploadlimit

import (
	"net/http"
	"testing"
)

func TestUploadLimitFromForm(t *testing.T) {
	uploadLimit, err := NewUploadLimitFromForm(true, 123)
	if err != nil {
		t.Fatalf("new upload limit from unlimited form: %v", err)
	}
	if !uploadLimit.IsUnlimited() {
		t.Fatal("expected unlimited upload limit")
	}

	uploadLimit, err = NewUploadLimitFromForm(false, 2)
	if err != nil {
		t.Fatalf("new upload limit from limited form: %v", err)
	}
	if uploadLimit.IsUnlimited() {
		t.Fatal("expected limited upload limit")
	}
	if uploadLimit.MiB() != 2 {
		t.Fatalf("expected 2 MiB, got %d", uploadLimit.MiB())
	}
	if uploadLimit.Bytes() != 2*bytesPerMiB {
		t.Fatalf("expected %d bytes, got %d", 2*bytesPerMiB, uploadLimit.Bytes())
	}
	if uploadLimit.LabelWithUnlimited("unlimited") != "2.0 MB (2 MiB)" {
		t.Fatalf("unexpected label: %q", uploadLimit.LabelWithUnlimited("unlimited"))
	}
}

func TestUploadLimitFromFormValidation(t *testing.T) {
	_, err := NewUploadLimitFromForm(false, 0)
	_ = requireHTTPErrorStatus(t, err, http.StatusBadRequest)
}

func TestUploadLimitFromBytes(t *testing.T) {
	uploadLimit, err := NewUploadLimitFromBytes(bytesPerMiB + 1)
	if err != nil {
		t.Fatalf("new upload limit from bytes: %v", err)
	}
	if uploadLimit.MiB() != 2 {
		t.Fatalf("expected ceil-converted 2 MiB, got %d", uploadLimit.MiB())
	}

	_, err = NewUploadLimitFromBytes(-1)
	_ = requireHTTPErrorStatus(t, err, http.StatusBadRequest)
}
