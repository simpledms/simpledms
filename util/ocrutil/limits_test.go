package ocrutil

import "testing"

func TestMaxFileSizeBytesDefaultWhenUnset(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Setenv(MaxFileSizeEnvVar, "")

	got := MaxFileSizeBytes()
	if got != DefaultMaxFileSizeBytes {
		t.Fatalf("expected default %d, got %d", DefaultMaxFileSizeBytes, got)
	}
}

func TestMaxFileSizeBytesFromEnv(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Setenv(MaxFileSizeEnvVar, "1048576")

	got := MaxFileSizeBytes()
	if got != 1048576 {
		t.Fatalf("expected env limit 1048576, got %d", got)
	}
}

func TestMaxFileSizeBytesFallsBackForInvalidValue(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Setenv(MaxFileSizeEnvVar, "invalid")

	got := MaxFileSizeBytes()
	if got != DefaultMaxFileSizeBytes {
		t.Fatalf("expected default %d, got %d", DefaultMaxFileSizeBytes, got)
	}
}

func TestMaxFileSizeBytesFallsBackForNonPositiveValue(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Setenv(MaxFileSizeEnvVar, "0")

	got := MaxFileSizeBytes()
	if got != DefaultMaxFileSizeBytes {
		t.Fatalf("expected default %d, got %d", DefaultMaxFileSizeBytes, got)
	}
}

func TestIsFileTooLarge(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Setenv(MaxFileSizeEnvVar, "100")

	if IsFileTooLarge(100) {
		t.Fatalf("expected file with exact limit not to be too large")
	}
	if !IsFileTooLarge(101) {
		t.Fatalf("expected file larger than limit to be too large")
	}
}

func TestSetUnsafeMaxFileSizeBytes(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Cleanup(func() {
		unsafeMaxFileSizeBytes = -1
	})

	SetUnsafeMaxFileSizeBytes(100)

	if MaxFileSizeBytes() != 100 {
		t.Fatalf("expected unsafe max file size to be 100")
	}
}

func TestSetUnsafeMaxFileSizeBytesFallsBackForInvalidValue(t *testing.T) {
	unsafeMaxFileSizeBytes = -1
	t.Cleanup(func() {
		unsafeMaxFileSizeBytes = -1
	})

	SetUnsafeMaxFileSizeBytes(0)

	if MaxFileSizeBytes() != DefaultMaxFileSizeBytes {
		t.Fatalf("expected default for invalid value")
	}
}
