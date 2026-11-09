package server

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

// StaticFileHandler serves individual files that are not meant for search results. http.FileServer
// alone renders a browsable index for every directory, so directories are answered like missing
// files.
type StaticFileHandler struct {
	filesFS    fs.FS
	fileServer http.Handler
}

func NewStaticFileHandler(filesFS fs.FS) *StaticFileHandler {
	return &StaticFileHandler{
		filesFS:    filesFS,
		fileServer: http.FileServer(http.FS(filesFS)),
	}
}

func (qq *StaticFileHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	setNoIndexHeader(rw)
	if qq.isDirectory(req.URL.Path) {
		http.NotFound(rw, req)
		return
	}

	qq.fileServer.ServeHTTP(rw, req)
}

func (qq *StaticFileHandler) isDirectory(urlPath string) bool {
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" {
		return true
	}

	info, err := fs.Stat(qq.filesFS, name)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Println(err)
		}
		// http.FileServer answers the failed lookup itself.
		return false
	}

	return info.IsDir()
}

func setNoIndexHeader(rw http.ResponseWriter) {
	rw.Header().Set("X-Robots-Tag", "noindex")
}
