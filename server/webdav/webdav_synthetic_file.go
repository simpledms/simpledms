package webdav

import (
	"io"
	"os"

	"golang.org/x/net/webdav"
)

type webDAVSyntheticFile struct {
	info    webDAVFileInfo
	entries []os.FileInfo
	offset  int
}

func newWebDAVDirectory(pathx webDAVPath) *webDAVSyntheticFile {
	if pathx.isRoot {
		return &webDAVSyntheticFile{
			info: webDAVFileInfo{name: "/", isDir: true},
			entries: []os.FileInfo{
				webDAVFileInfo{name: webDAVInbox, isDir: true},
			},
		}
	}
	return &webDAVSyntheticFile{info: webDAVFileInfo{name: webDAVInbox, isDir: true}}
}

func (qq *webDAVSyntheticFile) Close() error                                 { return nil }
func (qq *webDAVSyntheticFile) Read([]byte) (int, error)                     { return 0, io.EOF }
func (qq *webDAVSyntheticFile) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (qq *webDAVSyntheticFile) Stat() (os.FileInfo, error)                   { return qq.info, nil }
func (qq *webDAVSyntheticFile) Write(p []byte) (int, error)                  { return len(p), nil }
func (qq *webDAVSyntheticFile) Readdir(count int) ([]os.FileInfo, error) {
	if count <= 0 {
		entries := qq.entries[qq.offset:]
		qq.offset = len(qq.entries)
		return entries, nil
	}
	if qq.offset >= len(qq.entries) {
		return nil, io.EOF
	}
	end := qq.offset + count
	if end > len(qq.entries) {
		end = len(qq.entries)
	}
	entries := qq.entries[qq.offset:end]
	qq.offset = end
	return entries, nil
}

var _ webdav.File = (*webDAVSyntheticFile)(nil)
