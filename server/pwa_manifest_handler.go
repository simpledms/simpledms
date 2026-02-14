package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

const devPWANameEnvVar = "SIMPLEDMS_DEV_PWA_NAME"
const pwaManifestInternalCommentKey = "x_simpledms_comment"

type PWAManifestHandler struct {
	assetsFS            fs.FS
	devMode             bool
	cachedManifestBytes []byte
}

func NewPWAManifestHandler(assetsFS fs.FS, devMode bool) *PWAManifestHandler {
	return &PWAManifestHandler{
		assetsFS: assetsFS,
		devMode:  devMode,
	}
}

func (qq *PWAManifestHandler) Handler(rw http.ResponseWriter, _ *http.Request) {
	manifestBytes, err := qq.manifestBytes()
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/manifest+json")
	_, err = rw.Write(manifestBytes)
	if err != nil {
		log.Println(err)
	}
}

func (qq *PWAManifestHandler) manifestBytes() ([]byte, error) {
	if len(qq.cachedManifestBytes) > 0 {
		return qq.cachedManifestBytes, nil
	}

	manifestBytes, err := qq.loadManifestBytes()
	if err != nil {
		return nil, err
	}

	qq.cachedManifestBytes = manifestBytes

	return qq.cachedManifestBytes, nil
}

func (qq *PWAManifestHandler) loadManifestBytes() ([]byte, error) {
	manifestBytes, err := fs.ReadFile(qq.assetsFS, "manifest.json")
	if err != nil {
		return nil, err
	}

	manifestData := map[string]any{}
	err = json.Unmarshal(manifestBytes, &manifestData)
	if err != nil {
		return nil, err
	}

	delete(manifestData, pwaManifestInternalCommentKey)

	if qq.devMode {
		pwaName := strings.TrimSpace(os.Getenv(devPWANameEnvVar))
		if pwaName != "" {
			manifestData["name"] = pwaName
			manifestData["short_name"] = pwaName
		}
	}

	manifestBytes, err = json.Marshal(manifestData)
	if err != nil {
		return nil, err
	}

	return manifestBytes, nil
}
