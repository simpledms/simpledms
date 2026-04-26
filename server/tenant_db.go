package server

import (
	"fmt"

	tenantmodel "github.com/marcobeierer/go-core/model/tenant"
	"github.com/simpledms/simpledms/db/sqlx"
)

func openSimpleDMSTenantDBFromConfig(config tenantmodel.TenantDBOpenConfig) tenantmodel.TenantDB {
	tenantDB := sqlx.NewTenantDB(config.ReadOnlyDataSourceURL, config.ReadWriteDataSourceURL)
	if config.DevMode {
		tenantDB.Debug()
	}

	return tenantDB
}

func asSimpleDMSTenantDB(tenantDB tenantmodel.TenantDB) (*sqlx.TenantDB, error) {
	simpleDMSTenantDB, ok := tenantDB.(*sqlx.TenantDB)
	if !ok {
		return nil, fmt.Errorf("unexpected tenant db type %T", tenantDB)
	}

	return simpleDMSTenantDB, nil
}

func openSimpleDMSTenantDB(
	tenantm *tenantmodel.Tenant,
	devMode bool,
	metaPath string,
) (*sqlx.TenantDB, error) {
	tenantDB, err := tenantm.OpenDB(devMode, metaPath, func(config tenantmodel.TenantDBOpenConfig) (tenantmodel.TenantDB, error) {
		return openSimpleDMSTenantDBFromConfig(config), nil
	})
	if err != nil {
		return nil, err
	}

	return asSimpleDMSTenantDB(tenantDB)
}
