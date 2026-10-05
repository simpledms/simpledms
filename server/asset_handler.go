package server

import (
	"io/fs"
	"log"
	"net/http"
	"path"
	"regexp"

	"github.com/simpledms/simpledms/ui"
)

// esbuild names shared chunks after their content, so a changed chunk gets a new URL.
var contentHashedChunkName = regexp.MustCompile(`^chunk-[A-Z0-9]{8}\.js(\.map)?$`)

// AssetHandler serves the embedded assets with content-based ETags. Embedded files have no
// modification time, so without an ETag browsers download every asset on each page load.
type AssetHandler struct {
	assetVersions *ui.AssetVersions
	fileServer    http.Handler
}

func NewAssetHandler(assetsFS fs.FS, assetVersions *ui.AssetVersions) *AssetHandler {
	return &AssetHandler{
		assetVersions: assetVersions,
		fileServer:    http.FileServer(http.FS(assetsFS)),
	}
}

func (qq *AssetHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	hash, hasHash, err := qq.assetVersions.Hash(req.URL.Path)
	if err != nil {
		log.Println(err)
	}
	if hasHash {
		isVersioned := req.URL.Query().Get("v") == hash ||
			contentHashedChunkName.MatchString(path.Base(req.URL.Path))
		if isVersioned {
			// the URL changes with the content, see ui.AssetVersions
			rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// no-cache revalidates on every load, so a new deployment never serves stale assets;
			// unchanged assets are answered with 304 Not Modified by http.FileServer.
			rw.Header().Set("Cache-Control", "no-cache")
		}
		// Weak because the compression middleware may change the encoded response bytes.
		rw.Header().Set("ETag", `W/"`+hash+`"`)
	}

	qq.fileServer.ServeHTTP(rw, req)
}
