package ocrutil

import (
	"log"
	"os"
	"strconv"
)

const (
	MaxFileSizeMiBEnvVar        = "SIMPLEDMS_OCR_MAX_FILE_SIZE_MIB"
	DefaultMaxFileSizeMiB int64 = 25
	bytesPerMiB           int64 = 1024 * 1024
)

// unsafe because it should not be used directly
var unsafeMaxFileSizeMiB int64 = -1

func SetMaxFileSizeMiB(limit int64) {
	if limit <= 0 {
		unsafeMaxFileSizeMiB = DefaultMaxFileSizeMiB
		return
	}

	unsafeMaxFileSizeMiB = limit
}

func parseMaxFileSizeMiB(raw string) int64 {
	if raw == "" {
		return DefaultMaxFileSizeMiB
	}

	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || limit <= 0 {
		log.Println("invalid OCR max file size MiB env var, using default")
		return DefaultMaxFileSizeMiB
	}

	return limit
}

func MaxFileSizeMiB() int64 {
	if unsafeMaxFileSizeMiB >= 0 {
		return unsafeMaxFileSizeMiB
	}

	unsafeMaxFileSizeMiB = parseMaxFileSizeMiB(os.Getenv(MaxFileSizeMiBEnvVar))
	return unsafeMaxFileSizeMiB
}

func MaxFileSizeBytes() int64 {
	return MaxFileSizeMiB() * bytesPerMiB
}

func IsFileTooLarge(fileSizeBytes int64) bool {
	return fileSizeBytes > MaxFileSizeBytes()
}
