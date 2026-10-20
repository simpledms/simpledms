package scheduler

import (
	"context"
	"io/fs"
	"log"
	"runtime/debug"
	"time"

	"entgo.io/ent/privacy"
	"github.com/marcobeierer/go-tika"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/internal/gotenberg"
	"github.com/simpledms/simpledms/internal/xberg"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
)

const defaultSchedulerBatchSize = 100

// TODO rename to Runner?
type Scheduler struct {
	infra                  *common.Infra
	mainDB                 *sqlx.MainDB
	tenantDBs              *tenantdbs.TenantDBs
	s3Client               *minio.Client
	bucketName             string
	tikaClientNilable      *tika.Client
	xbergClientNilable     *xberg.XbergClient
	gotenbergClientNilable *gotenberg.GotenbergClient
	previewDiscoveryCursor map[*sqlx.TenantDB]int64
	lastOrphanObjectScan   time.Time
}

func NewScheduler(
	infra *common.Infra,
	mainDB *sqlx.MainDB,
	tenantDBs *tenantdbs.TenantDBs,
	s3Client *minio.Client,
	bucketName string,
	tikaClient *tika.Client,
	xbergClient *xberg.XbergClient,
	gotenbergClient *gotenberg.GotenbergClient,
) *Scheduler {
	return &Scheduler{
		infra:                  infra,
		mainDB:                 mainDB,
		tenantDBs:              tenantDBs,
		s3Client:               s3Client,
		bucketName:             bucketName,
		tikaClientNilable:      tikaClient,
		xbergClientNilable:     xbergClient,
		gotenbergClientNilable: gotenbergClient,
		previewDiscoveryCursor: make(map[*sqlx.TenantDB]int64),
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

	qq.runTenantDataMigrationsOnce(
		privacy.DecisionContext(context.Background(), privacy.Allow),
		tenantdatamigration.NewRunner(),
	)

	go qq.runRecovering("initialize tenants", time.Minute, func() {
		qq.initializeTenants(devMode, metaPath, migrationsTenantFS)
	})
	go qq.runRecovering("send mails", time.Minute, qq.sendMails)
	// runs once per start; only retried if it panics
	go qq.runRecovering("cleanup sessions", 5*time.Minute, qq.cleanupSessions)
	go qq.runRecovering("process files", time.Minute, qq.processFiles)
	go qq.runRecovering("apply OCR", time.Minute, qq.applyOCR)
	if qq.xbergClientNilable != nil {
		go qq.runRecovering("apply OCR markdown", time.Minute, qq.applyOCRMarkdown)
	}
	go qq.runRecovering("backfill content hashes", time.Minute, qq.backfillContentHashes)
	if qq.gotenbergClientNilable != nil {
		go qq.runRecovering("convert previews", time.Minute, qq.convertPreviews)
	}
}

// runRecovering runs a job and restarts it after recoverDelay whenever it panics. Each job
// recovers independently, so a panic in one loop never stops another loop mid-execution.
// It returns when the job returns normally.
//
// TODO what is a good recover delay?
func (qq *Scheduler) runRecovering(name string, recoverDelay time.Duration, job func()) {
	for runUntilPanic(name, job) {
		time.Sleep(recoverDelay)
	}
}

func runUntilPanic(name string, job func()) (hasPanicked bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover scheduler job:", name)
			hasPanicked = true
		}
	}()
	job()
	return false
}
