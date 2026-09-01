package webdav

import (
	"net/netip"
	"testing"
)

func TestParseTrustedProxyCIDRs(t *testing.T) {
	prefixes, err := ParseTrustedProxyCIDRs(" 192.0.2.10/32, 2001:db8::/32 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(prefixes) != 2 ||
		!prefixes[0].Contains(netip.MustParseAddr("192.0.2.10")) ||
		!prefixes[1].Contains(netip.MustParseAddr("2001:db8::1")) {
		t.Fatalf("unexpected trusted proxy prefixes: %v", prefixes)
	}
	if _, err := ParseTrustedProxyCIDRs("not-a-cidr"); err == nil {
		t.Fatal("expected invalid CIDR to fail")
	}
}
