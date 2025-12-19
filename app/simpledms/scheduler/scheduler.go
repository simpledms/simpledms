package scheduler

import (
	"io/fs"

	"github.com/marcobeierer/go-tika"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/app/simpledms/sqlx"
)

// TODO rename to Runner?
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

func (qq *Scheduler) Run(devMode bool, metaPath string, migrationsTenantFS fs.FS) {
	// IMPORTANT
	// recover must be implement independent of loop, otherwise an error in tenant loop
	// might stop mail loop, maybe in middle of execution

	// TODO use transactions instead of db directly?

	/*
		ctx := context.Background()
		if qq.mainDB.SystemConfig.Query().Where(systemconfig.InitializedAtNotNil()).CountX(ctx) == 0 {
			log.Println("App not initialized yet, scheduler not started")
			return
		}
	*/

	go func() {
		qq.initializeTenants(devMode, metaPath, migrationsTenantFS)
	}()

	go func() {
		qq.sendMails()
	}()

	go func() {
		qq.cleanupSessions()
	}()

	go func() {
		qq.processFiles()
	}()

	go func() {
		qq.applyOCR()
	}()
}
