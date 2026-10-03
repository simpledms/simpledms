package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type Request struct {
	*http.Request
}

func NewRequest(request *http.Request) *Request {
	return &Request{Request: request}
}

// ClientIP returns the address resolved through trusted proxies, falling back to the
// direct peer address.
func (qq *Request) ClientIP() string {
	if clientIP, ok := qq.Context().Value(clientIPContextKey{}).(netip.Addr); ok {
		return clientIP.String()
	}

	remoteAddr := strings.TrimSpace(qq.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteAddr = strings.TrimSpace(host)
	}
	if remoteAddr == "" {
		return "unknown"
	}
	return remoteAddr
}
