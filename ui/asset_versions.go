package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"log"
	"path"
	"strings"
	"sync"
)

// AssetVersions provides content hashes of the embedded assets. Page templates reference assets
// with their hash, so browsers can cache them indefinitely while a deployment still serves
// changed assets immediately.
type AssetVersions struct {
	assetsFS fs.FS
	hashes   sync.Map
}

func NewAssetVersions(assetsFS fs.FS) *AssetVersions {
	return &AssetVersions{
		assetsFS: assetsFS,
	}
}

// Hash returns the content hash of an asset; name is relative to the assets root.
func (qq *AssetVersions) Hash(name string) (string, bool, error) {
	name = path.Clean(strings.TrimPrefix(name, "/"))
	if cachedHash, ok := qq.hashes.Load(name); ok {
		return cachedHash.(string), true, nil
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
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:16])
	qq.hashes.Store(name, hash)

	return hash, true, nil
}

// URL versions an asset URL like /assets/tailwind.css with its content hash. Use it only for
// assets that no module imports by their plain path; the browser would otherwise load the
// module twice, as two instances.
func (qq *AssetVersions) URL(assetURL string) string {
	hash, ok, err := qq.Hash(strings.TrimPrefix(assetURL, "/assets/"))
	if err != nil {
		log.Println(err)
		return assetURL
	}
	if !ok {
		log.Printf("asset %s not found, serving it unversioned", assetURL)
		return assetURL
	}
	return assetURL + "?v=" + hash
}
