package webdav

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
)

// Config contains the dependencies required by the WebDAV endpoint.
type Config struct {
	MainDB    *sqlx.MainDB
	TenantDBs *tenantdbs.TenantDBs
	Infra     *common.Infra
	DevMode   bool
	MetaPath  string
	I18n      *i18n.I18n
}
