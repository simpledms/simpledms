package ocrutil

import (
	"os"
	"strconv"
)

const (
	MaxFileSizeEnvVar             = "SIMPLEDMS_OCR_MAX_FILE_SIZE_BYTES"
	DefaultMaxFileSizeBytes int64 = 25 * 1024 * 1024
)

// unsafe because it should not be used directly
var unsafeMaxFileSizeBytes int64 = -1

func MaxFileSizeBytes() int64 {
	if unsafeMaxFileSizeBytes >= 0 {
		return unsafeMaxFileSizeBytes
	}

	raw := os.Getenv(MaxFileSizeEnvVar)
	if raw == "" {
		unsafeMaxFileSizeBytes = DefaultMaxFileSizeBytes
		return unsafeMaxFileSizeBytes
	}

	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || limit <= 0 {
		return DefaultMaxFileSizeBytes
	}

	unsafeMaxFileSizeBytes = limit
	return unsafeMaxFileSizeBytes
}

func IsFileTooLarge(fileSizeBytes int64) bool {
	return fileSizeBytes > MaxFileSizeBytes()
}
