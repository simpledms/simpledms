package temporaryfile

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	"regexp"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
	"github.com/simpledms/simpledms/util/txx"
	"github.com/simpledms/simpledms/util/uploadx"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type UploadFromURLService struct {
	fileSystem              *filesystem.S3FileSystem
	allowLocalURLs          bool
	openCloudOrigin         string
	openCloudLinkPassword   string
	blockedDownloadPrefixes []netip.Prefix
	downloadFileFromURL     func(context.Context, string, string) (string, io.ReadCloser, error)
}

// OpenCloudURLSource selects the restricted OpenCloud public-link downloader.
const OpenCloudURLSource = "opencloud"

var openCloudShareTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func NewUploadFromURLService(
	fileSystem *filesystem.S3FileSystem,
	allowLocalURLs bool,
	openCloudOrigin string,
	openCloudLinkPassword string,
) *UploadFromURLService {
	service := &UploadFromURLService{
		fileSystem:            fileSystem,
		allowLocalURLs:        allowLocalURLs,
		openCloudOrigin:       strings.TrimSpace(openCloudOrigin),
		openCloudLinkPassword: openCloudLinkPassword,
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
	qq.downloadFileFromURL = func(ctx context.Context, rawURL string, _ string) (string, io.ReadCloser, error) {
		return downloadFile(ctx, rawURL)
	}
}

func (qq *UploadFromURLService) ValidateURLForSource(rawURL string, source string) (string, error) {
	if source != "" && source != OpenCloudURLSource {
		return "", e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported URL source.")
	}

	urlx, err := qq.parseURL(rawURL)
	if err != nil {
		return "", err
	}
	if source == OpenCloudURLSource {
		if err := qq.validateOpenCloudURL(urlx); err != nil {
			return "", err
		}
	}

	return urlx.String(), nil
}

func (qq *UploadFromURLService) UploadFromURL(ctx ctxx.Context, rawURL string, source string) (string, error) {
	filename, body, err := qq.downloadFileFromURL(ctx, rawURL, source)
	if err != nil {
		if source != OpenCloudURLSource {
			log.Println(err)
		}

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
		return qq.fileSystem.PrepareTemporaryAccountUploadWithSource(
			ctx,
			writeTx,
			filename,
			uploadToken,
			1,
			expiresAt,
			filesource.URLImport,
		)
	})
	if err != nil {
		return "", err
	}

	uploadResult, err := qq.fileSystem.UploadPreparedTemporaryAccountFile(ctx, body, prepared)
	if err != nil {
		uploadx.HandleTemporaryFileUploadFailure(ctx, qq.fileSystem, prepared, err, true)
		return "", err
	}

	_, err = txx.WithMainWriteTx(ctx, func(writeTx *entmain.Tx) (*struct{}, error) {
		return nil, qq.fileSystem.FinalizePreparedTemporaryAccountUpload(ctx, writeTx, prepared, uploadResult)
	})
	if err != nil {
		uploadx.HandleTemporaryFileUploadFailure(ctx, qq.fileSystem, prepared, err, true)
		return "", err
	}

	return uploadToken, nil
}

func (qq *UploadFromURLService) downloadFile(
	ctx context.Context,
	rawURL string,
	source string,
) (string, io.ReadCloser, error) {
	normalizedURL, err := qq.ValidateURLForSource(rawURL, source)
	if err != nil {
		log.Println(err)
		return "", nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, normalizedURL, nil)
	if err != nil {
		log.Println(err)
		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid URL.")
	}

	request.Header.Set("User-Agent", "SimpleDMS open-file/from-url")
	if source == OpenCloudURLSource {
		request.SetBasicAuth("public", qq.openCloudLinkPassword)
		request.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	httpClient := qq.newHTTPClient(source)
	response, err := httpClient.Do(request)
	if err != nil {
		if source == OpenCloudURLSource {
			logOpenCloudRequestFailure(err)
		} else {
			log.Println(err)
		}

		var httpErr *e.HTTPError
		if errors.As(err, &httpErr) {
			return "", nil, err
		}

		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Could not download file from URL.")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		if source == OpenCloudURLSource {
			logOpenCloudResponseFailure(response)
		} else {
			log.Println("failed to download file from URL, status code", response.StatusCode)
		}
		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Could not download file from URL.")
	}

	filename := qq.extractFilename(request.URL, response)
	if !filenamex.IsAllowed(filename) {
		_ = response.Body.Close()
		log.Println("invalid filename from url", filename)
		return "", nil, e.NewHTTPErrorf(http.StatusBadRequest, "Could not determine filename.")
	}

	return filename, response.Body, nil
}

func logOpenCloudRequestFailure(err error) {
	var unknownAuthority x509.UnknownAuthorityError
	var invalidCertificate x509.CertificateInvalidError
	var hostnameError x509.HostnameError
	var networkError net.Error

	switch {
	case errors.As(err, &unknownAuthority),
		errors.As(err, &invalidCertificate),
		errors.As(err, &hostnameError):
		log.Println("OpenCloud public-link request failed TLS certificate verification; verify the certificate trust chain and SIMPLEDMS_OPENCLOUD_ORIGIN hostname")
	case errors.As(err, &networkError) && networkError.Timeout():
		log.Println("OpenCloud public-link request timed out; verify SIMPLEDMS_OPENCLOUD_ORIGIN is reachable from the SimpleDMS backend")
	default:
		log.Println("OpenCloud public-link request failed before a response was received; verify the configured origin, network connectivity, and TLS setup")
	}
}

func logOpenCloudResponseFailure(response *http.Response) {
	requestID := strings.TrimSpace(response.Header.Get("X-Request-ID"))
	if len(requestID) > 128 || strings.ContainsAny(requestID, "\r\n") {
		requestID = ""
	}
	requestIDSuffix := ""
	if requestID != "" {
		requestIDSuffix = fmt.Sprintf("; OpenCloud request ID %q", requestID)
	}

	switch response.StatusCode {
	case http.StatusUnauthorized:
		log.Printf("OpenCloud rejected the public-link credentials (HTTP 401)%s; verify SIMPLEDMS_OPENCLOUD_PUBLIC_LINK_PASSWORD exactly matches the OpenCloud extension configuration and restart SimpleDMS after changing it", requestIDSuffix)
	case http.StatusForbidden:
		log.Printf("OpenCloud denied the public-link download (HTTP 403)%s; verify the link is a downloadable view permission", requestIDSuffix)
	case http.StatusNotFound, http.StatusGone:
		log.Printf("OpenCloud public link is missing, expired, or already revoked (HTTP %d)%s; create a new export", response.StatusCode, requestIDSuffix)
	case http.StatusTooManyRequests:
		log.Printf("OpenCloud rate-limited the public-link download (HTTP 429)%s; retry later", requestIDSuffix)
	default:
		if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
			log.Printf("OpenCloud attempted to redirect the public-link download (HTTP %d)%s; redirects are rejected to protect the shared password", response.StatusCode, requestIDSuffix)
			return
		}
		log.Printf("OpenCloud public-link download failed (HTTP %d)%s", response.StatusCode, requestIDSuffix)
	}
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

func (qq *UploadFromURLService) validateOpenCloudURL(target *url.URL) error {
	if qq.openCloudOrigin == "" || qq.openCloudLinkPassword == "" {
		return e.NewHTTPErrorf(http.StatusServiceUnavailable, "OpenCloud import is not configured.")
	}

	origin, err := url.Parse(qq.openCloudOrigin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" ||
		(origin.Path != "" && origin.Path != "/") {
		return e.NewHTTPErrorf(http.StatusServiceUnavailable, "OpenCloud import is not configured correctly.")
	}
	if origin.Scheme != "https" && !(qq.allowLocalURLs && origin.Scheme == "http" && isLocalHost(origin.Hostname())) {
		return e.NewHTTPErrorf(http.StatusServiceUnavailable, "OpenCloud import requires HTTPS.")
	}
	if target.Scheme != origin.Scheme || !strings.EqualFold(target.Host, origin.Host) ||
		target.RawQuery != "" || target.Fragment != "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud public link.")
	}

	const prefix = "/remote.php/dav/public-files/"
	escapedPath := target.EscapedPath()
	if !strings.HasPrefix(escapedPath, prefix) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud public link.")
	}
	parts := strings.SplitN(strings.TrimPrefix(escapedPath, prefix), "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud public link.")
	}
	token, err := url.PathUnescape(parts[0])
	if err != nil || !openCloudShareTokenPattern.MatchString(token) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud public link.")
	}

	return nil
}

func (qq *UploadFromURLService) ValidateOpenCloudCallbackOrigin(rawOrigin string) (string, error) {
	if strings.TrimSpace(rawOrigin) == "" {
		return "", e.NewHTTPErrorf(http.StatusBadRequest, "OpenCloud callback origin is required.")
	}
	target, err := url.Parse(rawOrigin)
	if err != nil || target.Host == "" || target.User != nil || target.RawQuery != "" ||
		target.Fragment != "" || (target.Path != "" && target.Path != "/") {
		return "", e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud callback origin.")
	}
	origin, err := url.Parse(qq.openCloudOrigin)
	if err != nil || target.Scheme != origin.Scheme || !strings.EqualFold(target.Host, origin.Host) {
		return "", e.NewHTTPErrorf(http.StatusBadRequest, "Invalid OpenCloud callback origin.")
	}
	return target.Scheme + "://" + target.Host, nil
}

func (qq *UploadFromURLService) newHTTPClient(source string) *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	useProxy := source != OpenCloudURLSource
	secureTransport := qq.newHTTPTransport(dialer, false, useProxy)
	var transport http.RoundTripper = secureTransport
	if qq.allowLocalURLs {
		localTransport := qq.newHTTPTransport(dialer, true, useProxy)
		transport = roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			if isLocalHost(request.URL.Hostname()) {
				return localTransport.RoundTrip(request)
			}

			return secureTransport.RoundTrip(request)
		})
	}

	timeout := 60 * time.Second
	if source == OpenCloudURLSource {
		timeout = 30 * time.Minute
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if source == OpenCloudURLSource {
				return http.ErrUseLastResponse
			}
			if len(via) >= 5 {
				return e.NewHTTPErrorf(http.StatusBadRequest, "Too many redirects.")
			}
			_, err := qq.parseURL(req.URL.String())
			return err
		},
	}
}

func (qq *UploadFromURLService) newHTTPTransport(
	dialer *net.Dialer,
	insecureSkipVerify bool,
	useProxy bool,
) *http.Transport {
	transport := &http.Transport{
		DialContext:           qq.safeDialContext(dialer),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	if useProxy {
		transport.Proxy = http.ProxyFromEnvironment
	}

	if insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{ // #nosec G402 -- restricted to loopback in dev mode.
			InsecureSkipVerify: true,
		}
	}

	return transport
}

func isLocalHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "localhost" {
		return true
	}

	addr, err := netip.ParseAddr(host)
	return err == nil && addr.Unmap().IsLoopback()
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
