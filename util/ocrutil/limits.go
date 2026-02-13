package ocrutil

import (
	"log"
	"os"
	"strconv"
)

const (
	MaxFileSizeEnvVar             = "SIMPLEDMS_OCR_MAX_FILE_SIZE_BYTES"
	DefaultMaxFileSizeBytes int64 = 25 * 1024 * 1024
)

// unsafe because it should not be used directly
var unsafeMaxFileSizeBytes int64 = -1

func SetUnsafeMaxFileSizeBytes(limit int64) {
	if limit <= 0 {
		unsafeMaxFileSizeBytes = DefaultMaxFileSizeBytes
		return
	}

	unsafeMaxFileSizeBytes = limit
}

func parseMaxFileSizeBytes(raw string) int64 {
	if raw == "" {
		return DefaultMaxFileSizeBytes
	}

	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || limit <= 0 {
		log.Println("invalid OCR max file size env var, using default")
		return DefaultMaxFileSizeBytes
	}

	return limit
}

func MaxFileSizeBytes() int64 {
	if unsafeMaxFileSizeBytes >= 0 {
		return unsafeMaxFileSizeBytes
	}

	unsafeMaxFileSizeBytes = parseMaxFileSizeBytes(os.Getenv(MaxFileSizeEnvVar))
	return unsafeMaxFileSizeBytes
}

func IsFileTooLarge(fileSizeBytes int64) bool {
	return fileSizeBytes > MaxFileSizeBytes()
}
