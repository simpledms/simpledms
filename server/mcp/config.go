package mcp

import (
	"net/netip"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
)

type Config struct {
	MainDB         *sqlx.MainDB
	TenantDBs      *tenantdbs.TenantDBs
	Infra          *common.Infra
	I18n           *i18n.I18n
	DevMode        bool
	TrustedProxies []netip.Prefix
}
