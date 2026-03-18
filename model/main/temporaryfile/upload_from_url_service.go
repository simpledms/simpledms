package temporaryfile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
	"github.com/simpledms/simpledms/util/txx"
	"github.com/simpledms/simpledms/util/uploadx"
)

type UploadFromURLService struct {
	fileSystem              *filesystem.S3FileSystem
	allowLocalURLs          bool
	blockedDownloadPrefixes []netip.Prefix
	downloadFileFromURL     func(context.Context, string) (string, io.ReadCloser, error)
}

func NewUploadFromURLService(
	fileSystem *filesystem.S3FileSystem,
	allowLocalURLs bool,
) *UploadFromURLService {
	service := &UploadFromURLService{
		fileSystem:     fileSystem,
		allowLocalURLs: allowLocalURLs,
		blockedDownloadPrefixes: []netip.Prefix{
			// Carrier-grade NAT range (RFC 6598).
			netip.MustParsePrefix("100.64.0.0/10"),
			// Benchmarking/testing range (RFC 2544).
			netip.MustParsePrefix("198.18.0.0/15"),
			// IPv4 multicast.
			netip.MustParsePrefix("224.0.0.0/4"),
			// Reserved IPv4 space.
			netip.MustParsePrefix("240.0.0.0/4"),
			// Unspecified IPv6 address.
			netip.MustParsePrefix("::/128"),
			// IPv6 documentation prefix.
			netip.MustParsePrefix("2001:db8::/32"),
			// IPv6 multicast.
			netip.MustParsePrefix("ff00::/8"),
		},
	}

	service.downloadFileFromURL = service.downloadFile

	return service
}

func (qq *UploadFromURLService) SetDownloadFileForTesting(
	downloadFile func(context.Context, string) (string, io.ReadCloser, error),
) {
	qq.downloadFileFromURL = downloadFile
}

func (qq *UploadFromURLService) ValidateURL(rawURL string) (string, error) {
	urlx, err := qq.parseURL(rawURL)
	if err != nil {
		return "", err
	}

	return urlx.String(), nil
}

func (qq *UploadFromURLService) UploadFromURL(ctx ctxx.Context, rawURL string) (string, error) {
	filename, body, err := qq.downloadFileFromURL(ctx, rawURL)
	if err != nil {
		log.Println(err)

		var httpErr *e.HTTPError
		if errors.As(err, &httpErr) {
			return "", err
		}

		return "", e.NewHTTPErrorf(http.StatusBadRequest, "Could not download file from URL.")
	}
	defer func() {
		if err := body.Close(); err != nil {
			log.Println(err)
		}
	}()

	uploadToken, err := qq.processDownloadedFile(ctx, filename, body)
	if err != nil {
		log.Println(err)

		var httpErr *e.HTTPError
		if errors.As(err, &httpErr) {
			return "", err
		}

		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "Processing of downloaded file failed.")
	}

	return uploadToken, nil
}

func (qq *UploadFromURLService) processDownloadedFile(
	ctx ctxx.Context,
	filename string,
	body io.Reader,
) (string, error) {
	uploadToken, err := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz_", 16)
	if err != nil {
		log.Println(err)
		return "", err
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	prepared, err := txx.WithMainWriteTx(ctx, func(writeTx *entmain.Tx) (*filesystem.PreparedAccountUpload, error) {
		return qq.fileSystem.PrepareTemporaryAccountUpload(
			ctx,
			writeTx,
			filename,
			uploadToken,
			1,
			expiresAt,
		)
	})
	if err != nil {
		return "", err
	}

	fileInfo, fileSize, err := qq.fileSystem.UploadPreparedTemporaryAccountFile(ctx, body, prepared)
	if err != nil {
		uploadx.HandleTemporaryFileUploadFailure(ctx, qq.fileSystem, prepared, err, true)
		return "", err
	}

	_, err = txx.WithMainWriteTx(ctx, func(writeTx *entmain.Tx) (*struct{}, error) {
		return nil, qq.fileSystem.FinalizePreparedTemporaryAccountUpload(ctx, writeTx, prepared, fileInfo, fileSize)
	})
	if err != nil {
		uploadx.HandleTemporaryFileUploadFailure(ctx, qq.fileSystem, prepared, err, false)
		return "", err
	}

	return uploadToken, nil
}

func (qq *UploadFromURLService) downloadFile(
	ctx context.Context,
	rawURL string,
) (string, io.ReadCloser, error) {
	urlx, err := qq.parseURL(rawURL)
	if err != nil {
		log.Println(err)
		return "", nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlx.String(), nil)
	if err != nil {
		log.Println(err)
		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid URL.")
	}

	request.Header.Set("User-Agent", "SimpleDMS open-file/from-url")

	httpClient := qq.newHTTPClient()
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err)

		var httpErr *e.HTTPError
		if errors.As(err, &httpErr) {
			return "", nil, err
		}

		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Could not download file from URL.")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		log.Println("failed to download file from URL, status code", response.StatusCode)
		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Could not download file from URL.")
	}

	filename := qq.extractFilename(urlx, response)
	if !filenamex.IsAllowed(filename) {
		_ = response.Body.Close()
		log.Println("invalid filename from url", filename)
		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Could not determine filename.")
	}

	return filename, response.Body, nil
}

func (qq *UploadFromURLService) parseURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "URL is required.")
	}

	urlx, err := url.Parse(rawURL)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid URL.")
	}

	if urlx.Scheme != "http" && urlx.Scheme != "https" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Only HTTP and HTTPS URLs are allowed.")
	}

	if urlx.Hostname() == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid URL.")
	}

	if urlx.User != nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "URL credentials are not allowed.")
	}

	host := strings.TrimSuffix(strings.ToLower(urlx.Hostname()), ".")
	// In dev mode we allow localhost-style imports for local integrations.
	if host == "localhost" && !qq.allowLocalURLs {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Local URLs are not allowed.")
	}

	if parsedAddr, err := netip.ParseAddr(host); err == nil {
		if qq.isBlockedDownloadAddr(parsedAddr) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Target host is not allowed.")
		}
	}

	return urlx, nil
}

func (qq *UploadFromURLService) newHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           qq.safeDialContext(dialer),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: time.Second,
	}

	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return e.NewHTTPErrorf(http.StatusBadRequest, "Too many redirects.")
			}
			_, err := qq.parseURL(req.URL.String())
			return err
		},
	}
}

func (qq *UploadFromURLService) safeDialContext(
	dialer *net.Dialer,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		if parsedAddr, err := netip.ParseAddr(host); err == nil {
			if qq.isBlockedDownloadAddr(parsedAddr) {
				return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Target host is not allowed.")
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(parsedAddr.String(), port))
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		var nilableLastErr error
		for _, ip := range ips {
			addr, ok := netip.AddrFromSlice(ip.IP)
			if !ok {
				continue
			}

			if qq.isBlockedDownloadAddr(addr) {
				nilableLastErr = e.NewHTTPErrorf(http.StatusBadRequest, "Target host is not allowed.")
				continue
			}

			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
			if err == nil {
				return conn, nil
			}

			nilableLastErr = err
		}

		if nilableLastErr != nil {
			return nil, nilableLastErr
		}

		return nil, errors.New("no reachable host address")
	}
}

func (qq *UploadFromURLService) isBlockedDownloadAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	// Dev mode only relaxes loopback. Private/link-local ranges remain blocked.
	if (!qq.allowLocalURLs && addr.IsLoopback()) ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() {
		return true
	}

	for _, prefix := range qq.blockedDownloadPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

func (qq *UploadFromURLService) extractFilename(urlx *url.URL, response *http.Response) string {
	contentDisposition := response.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			filename := strings.TrimSpace(params["filename"])
			if filename != "" {
				return filename
			}
		}
	}

	base := path.Base(urlx.Path)
	if base != "" && base != "." && base != "/" {
		decodedBase, err := url.PathUnescape(base)
		if err == nil {
			base = decodedBase
		}
		if base != "" {
			return base
		}
	}

	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if contentType != "" {
		extensions, err := mime.ExtensionsByType(strings.Split(contentType, ";")[0])
		if err == nil && len(extensions) > 0 {
			return fmt.Sprintf("download%s", extensions[0])
		}
	}

	return "download"
}
