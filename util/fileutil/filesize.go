package fileutil

import (
	"fmt"
)

const (
	_  = iota
	kB = 1 << (10 * iota)
	MB
	GB
	TB
)

// FormatSize formats the given size in bytes to a human-readable string
// with appropriate unit suffix (B, kB, MB, GB, TB)
func FormatSize(bytes int64) string {
	if bytes < kB {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < MB {
		return fmt.Sprintf("%.1f kB", float64(bytes)/float64(kB))
	}
	if bytes < GB {
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	}
	if bytes < TB {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	}
	return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
}
