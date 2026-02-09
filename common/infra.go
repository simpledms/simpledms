package common

import (
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/pluginx"
	"github.com/simpledms/simpledms/ui"
)

// TODO move to internal? was not possible because of circular deps...
type Infra struct {
	renderer       *ui.Renderer
	metaPath       string
	fileSystem     *filesystem.S3FileSystem // TODO is this a good location?
	factory        *Factory
	FileRepo       *FileRepository
	pluginRegistry *pluginx.Registry
	// nilableMainIdentity *age.X25519Identity
	systemConfig *modelmain.SystemConfig
	// no minio.Client, seems to risky for misuse; inject on demand
	// same is true for db clients
}

func NewInfra(
	renderer *ui.Renderer,
	metaPath string,
	fileSystem *filesystem.S3FileSystem,
	factory *Factory,
	fileRepo *FileRepository,
	pluginRegistry *pluginx.Registry,
	// nilableMainIdentity *age.X25519Identity,
	systemConfig *modelmain.SystemConfig,
) *Infra {
	return &Infra{
		renderer:       renderer,
		metaPath:       metaPath,
		fileSystem:     fileSystem,
		factory:        factory,
		FileRepo:       fileRepo,
		pluginRegistry: pluginRegistry,
		systemConfig:   systemConfig,
		// nilableMainIdentity: nilableMainIdentity,
	}
}

/*
// unsafe because usually you want to use TTx
func (qq *Infra) UnsafeDB() *enttenant.Client {
	return qq.db
}
*/

func (qq *Infra) Renderer() *ui.Renderer {
	return qq.renderer
}

func (qq *Infra) FileSystem() *filesystem.S3FileSystem {
	return qq.fileSystem
}

func (qq *Infra) Factory() *Factory {
	return qq.factory
}

func (qq *Infra) SystemConfig() *modelmain.SystemConfig {
	return qq.systemConfig
}

func (qq *Infra) PluginRegistry() *pluginx.Registry {
	return qq.pluginRegistry
}

/*func (qq *Infra) NilableMainIdentity() *age.X25519Identity {
	return qq.nilableMainIdentity
}*/
