package scheduler

import (
	"context"
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

const (
	contentHashBackfillEnabledEnv        = "CONTENT_HASH_BACKFILL_ENABLED"
	contentHashBackfillMaxFilesPerRunEnv = "CONTENT_HASH_BACKFILL_MAX_FILES_PER_RUN"
	contentHashBackfillMaxBytesPerRunEnv = "CONTENT_HASH_BACKFILL_MAX_BYTES_PER_RUN"
	contentHashBackfillMaxDurationEnv    = "CONTENT_HASH_BACKFILL_MAX_DURATION"
	contentHashBackfillIntervalEnv       = "CONTENT_HASH_BACKFILL_INTERVAL"
)

type contentHashBackfillSchedulerConfig struct {
	Enabled  bool
	Interval time.Duration
	Backfill filesystem.ContentHashBackfillConfig
}

func (qq *Scheduler) backfillContentHashes() {
	config := contentHashBackfillConfigFromEnv()
	if !config.Enabled {
		log.Println("content hash backfill disabled")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")
			time.Sleep(1 * time.Minute)
			qq.backfillContentHashes()
		}
	}()

	for {
		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		qq.backfillContentHashesOnce(ctx, config.Backfill)

		time.Sleep(config.Interval)
	}
}

func (qq *Scheduler) backfillContentHashesOnce(
	ctx context.Context,
	config filesystem.ContentHashBackfillConfig,
) {
	service := filesystem.NewContentHashBackfillService(qq.infra.FileSystem())

	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		tenantx, err := qq.mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.ID(tenantID)).Only(ctx)
		if err != nil {
			if entmain.IsNotFound(err) {
				log.Println("tenant not found", tenantID)
				return true
			}
			log.Println(err)
			return true
		}

		result, err := service.RunTenant(
			ctx,
			tenantDB.ReadWriteConn,
			tenantx.X25519IdentityEncrypted.Identity(),
			config,
		)
		if err != nil {
			log.Println(err)
			return true
		}
		if result.ProcessedFiles > 0 {
			log.Printf(
				"content hash backfill processed %d files and %d bytes for tenant %d",
				result.ProcessedFiles,
				result.ProcessedBytes,
				tenantID,
			)
		}

		return true
	})
}

func contentHashBackfillConfigFromEnv() contentHashBackfillSchedulerConfig {
	return contentHashBackfillSchedulerConfig{
		Enabled:  envBool(contentHashBackfillEnabledEnv, true),
		Interval: envDuration(contentHashBackfillIntervalEnv, 5*time.Minute),
		Backfill: filesystem.ContentHashBackfillConfig{
			MaxFilesPerRun: envInt(contentHashBackfillMaxFilesPerRunEnv, 1),
			MaxBytesPerRun: envInt64(contentHashBackfillMaxBytesPerRunEnv, 256*1024*1024),
			MaxDuration:    envDuration(contentHashBackfillMaxDurationEnv, 30*time.Second),
		},
	}
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Printf("invalid bool value %q for %s", value, name)
		return fallback
	}
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid int value %q for %s: %v", value, name, err)
		return fallback
	}
	return parsed
}

func envInt64(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("invalid int64 value %q for %s: %v", value, name, err)
		return fallback
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid duration value %q for %s: %v", value, name, err)
		return fallback
	}
	return parsed
}
