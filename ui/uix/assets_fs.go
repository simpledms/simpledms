package uix

import (
	"embed"
	"io/fs"
)

//go:embed web/assets
var assetsFS embed.FS

func NewAssetsFS() (fs.FS, error) {
	return fs.Sub(assetsFS, "web/assets")
}
