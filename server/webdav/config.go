package webdav

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
)

// Config contains the dependencies required by the WebDAV endpoint.
type Config struct {
	MainDB         *sqlx.MainDB
	TenantDBs      *tenantdbs.TenantDBs
	Infra          *common.Infra
	DevMode        bool
	MetaPath       string
	I18n           *i18n.I18n
	TrustedProxies []netip.Prefix
}

// ParseTrustedProxyCIDRs parses a comma-separated trusted proxy allowlist.
func ParseTrustedProxyCIDRs(value string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for cidr := range strings.SplitSeq(value, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy CIDR %q: %w", cidr, err)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}
