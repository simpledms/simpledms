package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	"filippo.io/age"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
)

type ContentHashBackfillConfig struct {
	MaxFilesPerRun int
	MaxBytesPerRun int64
	MaxDuration    time.Duration
}

type ContentHashBackfillResult struct {
	ProcessedFiles int
	ProcessedBytes int64
}

type ContentHashBackfillService struct {
	fileSystem *S3FileSystem
}

func NewContentHashBackfillService(fileSystem *S3FileSystem) *ContentHashBackfillService {
	return &ContentHashBackfillService{
		fileSystem: fileSystem,
	}
}

func (qq *ContentHashBackfillService) RunTenant(
	ctx context.Context,
	tenantClient *enttenant.Client,
	tenantIdentity *age.X25519Identity,
	config ContentHashBackfillConfig,
) (*ContentHashBackfillResult, error) {
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

	if config.MaxFilesPerRun <= 0 {
		config.MaxFilesPerRun = 1
	}

	files, err := tenantClient.StoredFile.Query().
		Where(
			storedfile.ContentSha256IsNil(),
			storedfile.UploadSucceededAtNotNil(),
			storedfile.CopiedToFinalDestinationAtNotNil(),
		).
		Order(storedfile.ByID(sql.OrderAsc())).
		Limit(config.MaxFilesPerRun).
		All(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	result := &ContentHashBackfillResult{}
	startedAt := time.Now()
	for _, filex := range files {
		if config.MaxDuration > 0 && time.Since(startedAt) >= config.MaxDuration {
			break
		}
		if qq.exceedsByteBudget(result.ProcessedBytes, filex.Size, config.MaxBytesPerRun) {
			break
		}

		contentSHA256, processedBytes, err := qq.calculateContentSHA256(ctx, tenantIdentity, filex)
		if err != nil {
			log.Println(err)
			continue
		}

		err = tenantClient.StoredFile.Update().
			Where(
				storedfile.ID(filex.ID),
				storedfile.ContentSha256IsNil(),
			).
			SetContentSha256(contentSHA256).
			Exec(ctx)
		if err != nil {
			log.Println(err)
			continue
		}

		result.ProcessedFiles++
		result.ProcessedBytes += processedBytes
	}

	return result, nil
}

func (qq *ContentHashBackfillService) exceedsByteBudget(
	processedBytes int64,
	fileSize int64,
	maxBytesPerRun int64,
) bool {
	if maxBytesPerRun <= 0 {
		return false
	}

	return processedBytes+fileSize > maxBytesPerRun
}

func (qq *ContentHashBackfillService) calculateContentSHA256(
	ctx context.Context,
	tenantIdentity *age.X25519Identity,
	filex *enttenant.StoredFile,
) (string, int64, error) {
	openedFile, err := qq.fileSystem.UnsafeOpenFile(
		ctx,
		tenantIdentity,
		storedfilemodel.NewStoredFile(filex),
	)
	if err != nil {
		log.Println(err)
		return "", 0, err
	}
	defer func() {
		if err := openedFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	hasher := sha256.New()
	processedBytes, err := io.Copy(hasher, openedFile)
	if err != nil {
		log.Println(err)
		return "", 0, err
	}

	return hex.EncodeToString(hasher.Sum(nil)), processedBytes, nil
}
