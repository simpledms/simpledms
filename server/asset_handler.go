package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
)

// AssetHandler serves the embedded assets with content-based ETags. Embedded files have no
// modification time, so without an ETag browsers download every asset on each page load.
type AssetHandler struct {
	assetsFS   fs.FS
	fileServer http.Handler
	etags      sync.Map
}

func NewAssetHandler(assetsFS fs.FS) *AssetHandler {
	return &AssetHandler{
		assetsFS:   assetsFS,
		fileServer: http.FileServer(http.FS(assetsFS)),
	}
}

func (qq *AssetHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	etag, hasETag, err := qq.etag(req.URL.Path)
	if err != nil {
		log.Println(err)
	}
	if hasETag {
		// no-cache revalidates on every load, so a new deployment never serves stale assets;
		// unchanged assets are answered with 304 Not Modified by http.FileServer.
		rw.Header().Set("Cache-Control", "no-cache")
		rw.Header().Set("ETag", etag)
	}

	qq.fileServer.ServeHTTP(rw, req)
}

func (qq *AssetHandler) etag(urlPath string) (string, bool, error) {
	name := path.Clean(strings.TrimPrefix(urlPath, "/"))
	if cachedETag, ok := qq.etags.Load(name); ok {
		return cachedETag.(string), true, nil
	}

	info, err := fs.Stat(qq.assetsFS, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	if info.IsDir() {
		return "", false, nil
	}

	content, err := fs.ReadFile(qq.assetsFS, name)
	if err != nil {
		return "", false, err
	}
	hash := sha256.Sum256(content)
	// Weak because the compression middleware may change the encoded response bytes.
	etag := `W/"` + hex.EncodeToString(hash[:16]) + `"`
	qq.etags.Store(name, etag)

	return etag, true, nil
}
