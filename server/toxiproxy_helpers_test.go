package server

import (
	"fmt"
	"strings"
	"testing"

	toxiproxy "github.com/Shopify/toxiproxy/client"
)

const (
	toxiproxyAddrEnv      = "SIMPLEDMS_TOXIPROXY_ADDR"
	toxiproxyUpstreamEnv  = "SIMPLEDMS_TOXIPROXY_UPSTREAM"
	toxiproxyListenPort   = "7071"
	toxiproxyListenHost   = "localhost"
	toxiproxyListenTarget = "0.0.0.0:7071"
)

func newS3Toxiproxy(t *testing.T) (*toxiproxy.Proxy, string) {
	t.Helper()

	client := toxiproxy.NewClient(envOrDefault(toxiproxyAddrEnv, "localhost:8474"))
	proxies, err := client.Proxies()
	if err != nil {
		t.Skipf("toxiproxy not available: %v", err)
	}

	for _, proxy := range proxies {
		if !strings.HasSuffix(proxy.Listen, ":"+toxiproxyListenPort) {
			continue
		}
		if !strings.HasPrefix(proxy.Name, "simpledms-s3-") {
			t.Skipf("toxiproxy listen port %s already in use by proxy %q", toxiproxyListenPort, proxy.Name)
		}
		if err := proxy.Delete(); err != nil {
			t.Fatalf("delete existing toxiproxy %s: %v", proxy.Name, err)
		}
	}

	proxyName := "simpledms-s3-" + sanitizeProxyName(t.Name())
	upstream := envOrDefault(toxiproxyUpstreamEnv, "versitygw:7070")
	proxy, err := client.CreateProxy(proxyName, toxiproxyListenTarget, upstream)
	if err != nil {
		t.Fatalf("create toxiproxy: %v", err)
	}

	t.Cleanup(func() {
		_ = proxy.Delete()
	})

	return proxy, fmt.Sprintf("%s:%s", toxiproxyListenHost, toxiproxyListenPort)
}

func sanitizeProxyName(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		" ", "-",
		":", "-",
		".", "-",
		"_", "-",
		"\\", "-",
	)
	return strings.ToLower(replacer.Replace(name))
}
