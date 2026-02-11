package ocrutil

import (
	"os"
	"strconv"
)

const (
	MaxFileSizeEnvVar             = "SIMPLEDMS_OCR_MAX_FILE_SIZE_BYTES"
	DefaultMaxFileSizeBytes int64 = 25 * 1024 * 1024
)

var defaultMaxFileSizeBytes int64 = -1

func MaxFileSizeBytes() int64 {
	if defaultMaxFileSizeBytes >= 0 {
		return defaultMaxFileSizeBytes
	}

	raw := os.Getenv(MaxFileSizeEnvVar)
	if raw == "" {
		defaultMaxFileSizeBytes = DefaultMaxFileSizeBytes
		return defaultMaxFileSizeBytes
	}

	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || limit <= 0 {
		return DefaultMaxFileSizeBytes
	}

	defaultMaxFileSizeBytes = limit
	return defaultMaxFileSizeBytes
}

func IsFileTooLarge(fileSizeBytes int64) bool {
	return fileSizeBytes > MaxFileSizeBytes()
}
