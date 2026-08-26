package filesystem

import "time"

type ContentHashBackfillConfig struct {
	MaxFilesPerRun int
	MaxBytesPerRun int64
	MaxDuration    time.Duration
}
