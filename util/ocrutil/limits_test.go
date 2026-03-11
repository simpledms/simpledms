package ocrutil

import "testing"

func TestMaxFileSizeBytesDefaultWhenUnset(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Setenv(MaxFileSizeMiBEnvVar, "")

	got := MaxFileSizeBytes()
	if got != DefaultMaxFileSizeMiB*bytesPerMiB {
		t.Fatalf("expected default %d, got %d", DefaultMaxFileSizeMiB*bytesPerMiB, got)
	}
}

func TestMaxFileSizeMiBFromEnv(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Setenv(MaxFileSizeMiBEnvVar, "3")

	got := MaxFileSizeMiB()
	if got != 3 {
		t.Fatalf("expected env limit 3, got %d", got)
	}
}

func TestMaxFileSizeMiBFallsBackForInvalidValue(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Setenv(MaxFileSizeMiBEnvVar, "invalid")

	got := MaxFileSizeMiB()
	if got != DefaultMaxFileSizeMiB {
		t.Fatalf("expected default %d, got %d", DefaultMaxFileSizeMiB, got)
	}
}

func TestMaxFileSizeMiBFallsBackForNonPositiveValue(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Setenv(MaxFileSizeMiBEnvVar, "0")

	got := MaxFileSizeMiB()
	if got != DefaultMaxFileSizeMiB {
		t.Fatalf("expected default %d, got %d", DefaultMaxFileSizeMiB, got)
	}
}

func TestIsFileTooLarge(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Setenv(MaxFileSizeMiBEnvVar, "1")

	if IsFileTooLarge(bytesPerMiB) {
		t.Fatalf("expected file with exact limit not to be too large")
	}
	if !IsFileTooLarge(bytesPerMiB + 1) {
		t.Fatalf("expected file larger than limit to be too large")
	}
}

func TestSetUnsafeMaxFileSizeMiB(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Cleanup(func() {
		unsafeMaxFileSizeMiB = -1
	})

	SetMaxFileSizeMiB(3)

	if MaxFileSizeMiB() != 3 {
		t.Fatalf("expected unsafe max file size to be 3")
	}
}

func TestSetUnsafeMaxFileSizeMiBFallsBackForInvalidValue(t *testing.T) {
	unsafeMaxFileSizeMiB = -1
	t.Cleanup(func() {
		unsafeMaxFileSizeMiB = -1
	})

	SetMaxFileSizeMiB(0)

	if MaxFileSizeMiB() != DefaultMaxFileSizeMiB {
		t.Fatalf("expected default for invalid value")
	}
}
