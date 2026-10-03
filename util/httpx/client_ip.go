package httpx

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// ClientIPThroughTrustedProxies returns the direct peer address, or, if the peer is a
// trusted proxy, the rightmost X-Forwarded-For address that is not a trusted proxy.
// Entries left of that address are client-controlled and must be ignored.
func ClientIPThroughTrustedProxies(
	req *http.Request,
	trustedProxies []netip.Prefix,
) (netip.Addr, bool) {
	directAddr, ok := parseClientAddr(req.RemoteAddr)
	if !ok {
		return netip.Addr{}, false
	}
	if !isTrustedProxyAddr(directAddr, trustedProxies) {
		return directAddr, true
	}

	// Proxies either append to the client's header line or add their own line, so the
	// trusted entries are at the end of all lines combined, not of the first line.
	forwardedFor := strings.Split(strings.Join(req.Header.Values("X-Forwarded-For"), ","), ",")
	for _, forwardedAddr := range slices.Backward(forwardedFor) {
		addr, ok := parseClientAddr(forwardedAddr)
		if !ok {
			return directAddr, true
		}
		if !isTrustedProxyAddr(addr, trustedProxies) {
			return addr, true
		}
	}
	return directAddr, true
}

// parseClientAddr accepts an address with or without port and unmaps IPv4-mapped IPv6.
func parseClientAddr(value string) (netip.Addr, bool) {
	value = strings.TrimSpace(value)
	if addrPort, err := netip.ParseAddrPort(value); err == nil {
		return addrPort.Addr().Unmap(), true
	}
	addr, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func isTrustedProxyAddr(addr netip.Addr, trustedProxies []netip.Prefix) bool {
	mappedAddr := netip.AddrFrom16(addr.As16())
	for _, prefix := range trustedProxies {
		if prefix.Contains(addr) || prefix.Contains(mappedAddr) {
			return true
		}
	}
	return false
}
