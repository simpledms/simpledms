package scheduler

import (
	"github.com/marcobeierer/go-tika"
	"github.com/minio/minio-go/v7"

	"github.com/marcobeierer/go-core/db/sqlx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
)

type Scheduler struct {
	infra             *common.Infra
	mainDB            *sqlx.MainDB
	tenantDBs         *tenantdbs.TenantDBs
	s3Client          *minio.Client
	bucketName        string
	tikaClientNilable *tika.Client
}

func NewScheduler(
	infra *common.Infra,
	mainDB *sqlx.MainDB,
	tenantDBs *tenantdbs.TenantDBs,
	s3Client *minio.Client,
	bucketName string,
	tikaClient *tika.Client,
) *Scheduler {
	return &Scheduler{
		infra:             infra,
		mainDB:            mainDB,
		tenantDBs:         tenantDBs,
		s3Client:          s3Client,
		bucketName:        bucketName,
		tikaClientNilable: tikaClient,
	}
}

func (qq *Scheduler) Run() {
	go func() {
		qq.processFiles()
	}()

	go func() {
		qq.applyOCR()
	}()
}
