package common

import (
	corecommon "github.com/marcobeierer/go-core/common"
	systemconfigmodel "github.com/marcobeierer/go-core/model/systemconfig"
	"github.com/marcobeierer/go-core/pluginx"
	"github.com/marcobeierer/go-core/ui"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

type Infra struct {
	*corecommon.Infra
	fileSystem *filesystem.S3FileSystem
	FileRepo   *FileRepository
}

func NewInfra(
	renderer *ui.Renderer,
	metaPath string,
	fileSystem *filesystem.S3FileSystem,
	fileRepo *FileRepository,
	pluginRegistry *pluginx.Registry,
	systemConfig *systemconfigmodel.SystemConfig,
) *Infra {
	return &Infra{
		Infra: corecommon.NewInfra(
			renderer,
			metaPath,
			pluginRegistry,
			systemConfig,
		),
		fileSystem: fileSystem,
		FileRepo:   fileRepo,
	}
}

func (qq *Infra) CoreInfra() *corecommon.Infra {
	return qq.Infra
}

func (qq *Infra) FileSystem() *filesystem.S3FileSystem {
	return qq.fileSystem
}
