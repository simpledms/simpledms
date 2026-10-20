package common

import (
	systemconfigmodel "github.com/simpledms/simpledms/model/main/systemconfig"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/pluginx"
	"github.com/simpledms/simpledms/ui"
)

// TODO move to internal? was not possible because of circular deps...
type Infra struct {
	renderer                             *ui.Renderer
	metaPath                             string
	fileSystem                           *filesystem.S3FileSystem // TODO is this a good location?
	FileRepo                             *FileRepository
	pluginRegistry                       *pluginx.Registry
	manageTenantsDeleteTenantCmdEndpoint string
	manageTenantsDownloadBackupEndpoint  string
	systemConfig                         *systemconfigmodel.SystemConfig
	// no minio.Client, seems to risky for misuse; inject on demand
	// same is true for db clients
}

func NewInfra(
	renderer *ui.Renderer,
	metaPath string,
	fileSystem *filesystem.S3FileSystem,
	_ *Factory, // deprecated, see Factory
	fileRepo *FileRepository,
	pluginRegistry *pluginx.Registry,
	systemConfig *systemconfigmodel.SystemConfig,
) *Infra {
	return &Infra{
		renderer:                             renderer,
		metaPath:                             metaPath,
		fileSystem:                           fileSystem,
		FileRepo:                             fileRepo,
		pluginRegistry:                       pluginRegistry,
		manageTenantsDeleteTenantCmdEndpoint: "",
		manageTenantsDownloadBackupEndpoint:  "",
		systemConfig:                         systemConfig,
	}
}

func (qq *Infra) Renderer() *ui.Renderer {
	return qq.renderer
}

func (qq *Infra) MetaPath() string {
	return qq.metaPath
}

func (qq *Infra) FileSystem() *filesystem.S3FileSystem {
	return qq.fileSystem
}

func (qq *Infra) SystemConfig() *systemconfigmodel.SystemConfig {
	return qq.systemConfig
}

func (qq *Infra) PluginRegistry() *pluginx.Registry {
	return qq.pluginRegistry
}

func (qq *Infra) ManageTenantsDeleteTenantCmdEndpoint() string {
	return qq.manageTenantsDeleteTenantCmdEndpoint
}

func (qq *Infra) SetManageTenantsDeleteTenantCmdEndpoint(endpoint string) {
	qq.manageTenantsDeleteTenantCmdEndpoint = endpoint
}

func (qq *Infra) ManageTenantsDownloadBackupEndpoint() string {
	return qq.manageTenantsDownloadBackupEndpoint
}

func (qq *Infra) SetManageTenantsDownloadBackupEndpoint(endpoint string) {
	qq.manageTenantsDownloadBackupEndpoint = endpoint
}
