package webdav

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/simpledms/simpledms/util/filenamex"
)

const webDAVInbox = "Inbox"

type webDAVPath struct {
	canonical string
	filename  string
	isRoot    bool
	isInbox   bool
	isFile    bool
}

func parseWebDAVPath(req *http.Request, endpointPrefix string) (webDAVPath, bool) {
	if hasEncodedSeparator(req.URL.EscapedPath()) {
		return webDAVPath{}, false
	}
	if !strings.HasPrefix(req.URL.Path, endpointPrefix) {
		return webDAVPath{}, false
	}
	return parseWebDAVSuffix(strings.TrimPrefix(req.URL.Path, endpointPrefix))
}

func parseWebDAVSuffix(suffix string) (webDAVPath, bool) {
	if suffix == "" || suffix == "/" {
		return webDAVPath{canonical: "/", isRoot: true}, true
	}
	if strings.Contains(suffix, "\\") || strings.ContainsRune(suffix, 0) {
		return webDAVPath{}, false
	}
	for _, r := range suffix {
		if unicode.IsControl(r) {
			return webDAVPath{}, false
		}
	}
	if suffix == "/"+webDAVInbox || suffix == "/"+webDAVInbox+"/" {
		return webDAVPath{canonical: "/" + webDAVInbox + "/", isInbox: true}, true
	}
	if !strings.HasPrefix(suffix, "/"+webDAVInbox+"/") || strings.HasSuffix(suffix, "/") {
		return webDAVPath{}, false
	}

	filename := strings.TrimPrefix(suffix, "/"+webDAVInbox+"/")
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return webDAVPath{}, false
	}
	if !utf8.ValidString(filename) || path.Clean(filename) != filename || !filenamex.IsAllowed(filename) {
		return webDAVPath{}, false
	}

	return webDAVPath{
		canonical: "/" + webDAVInbox + "/" + filename,
		filename:  filename,
		isFile:    true,
	}, true
}

func parseWebDAVDestination(req *http.Request, endpointPrefix string) (webDAVPath, bool) {
	destination := strings.TrimSpace(req.Header.Get("Destination"))
	if destination == "" {
		return webDAVPath{}, false
	}
	destinationURL, err := url.Parse(destination)
	if err != nil {
		return webDAVPath{}, false
	}
	if destinationURL.Host != "" && !strings.EqualFold(destinationURL.Host, req.Host) {
		return webDAVPath{}, false
	}
	clone := new(http.Request)
	*clone = *req
	clone.URL = destinationURL
	if !destinationURL.IsAbs() {
		clone.URL = &url.URL{Path: destinationURL.Path, RawPath: destinationURL.RawPath}
	}
	return parseWebDAVPath(clone, endpointPrefix)
}

func hasEncodedSeparator(escapedPath string) bool {
	escapedPath = strings.ToLower(escapedPath)
	return strings.Contains(escapedPath, "%2f") || strings.Contains(escapedPath, "%5c")
}
