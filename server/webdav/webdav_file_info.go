package webdav

import (
	"os"
	"time"
)

type webDAVFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (qq webDAVFileInfo) Name() string { return qq.name }
func (qq webDAVFileInfo) Size() int64  { return qq.size }
func (qq webDAVFileInfo) Mode() os.FileMode {
	if qq.isDir {
		return os.ModeDir | 0555
	}
	return 0444
}
func (qq webDAVFileInfo) ModTime() time.Time { return time.Unix(0, 0).UTC() }
func (qq webDAVFileInfo) IsDir() bool        { return qq.isDir }
func (qq webDAVFileInfo) Sys() any           { return nil }
