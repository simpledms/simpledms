package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	_ "github.com/mattn/go-sqlite3"

	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	_ "github.com/simpledms/simpledms/db/entmain/runtime"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	_ "github.com/simpledms/simpledms/db/enttenant/runtime"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	tenantm "github.com/simpledms/simpledms/model/tenant"
	"github.com/simpledms/simpledms/pathx"
	"github.com/simpledms/simpledms/util"
)

func main() {
	metaPath := flag.String("meta", "simpledms", "Path to SimpleDMS data directory")
	tenantPublicID := flag.String("tenant", "", "Tenant public ID (default: first initialized tenant)")
	count := flag.Int("count", 1000000, "Number of files to create")
	batchSize := flag.Int("batch-size", 1000, "Files per transaction batch")
	minVersions := flag.Int("min-versions", 1, "Minimum file versions per file")
	maxVersions := flag.Int("max-versions", 5, "Maximum file versions per file")
	prefix := flag.String("prefix", "bench1m", "Prefix for generated file names")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed for version distribution")
	yes := flag.Bool("yes", false, "Skip confirmation prompt")
	flag.Parse()

	if *count < 0 {
		log.Fatalln("count must be >= 0")
	}
	if *batchSize <= 0 {
		log.Fatalln("batch-size must be > 0")
	}
	if *minVersions <= 0 {
		log.Fatalln("min-versions must be > 0")
	}
	if *maxVersions < *minVersions {
		log.Fatalln("max-versions must be >= min-versions")
	}

	mainDB := openMainDB(*metaPath)
	defer func() {
		if err := mainDB.Close(); err != nil {
			log.Println(err)
		}
	}()

	tenantx := resolveTenant(mainDB, *tenantPublicID)
	tenantDB := openTenantDB(tenantx, *metaPath)
	defer func() {
		if err := tenantDB.Close(); err != nil {
			log.Println(err)
		}
	}()

	estimatedVersionCount := int64(*count) * int64(*minVersions+*maxVersions) / 2
	if err := confirmRun(*yes, tenantx.PublicID.String(), *count, estimatedVersionCount, *prefix); err != nil {
		log.Fatalln(err)
	}

	spaceName := fmt.Sprintf("benchmark-%s-%d", *prefix, time.Now().Unix())
	spaceID, spacePublicID, rootDirID := createSpaceAndRootDir(tenantDB, spaceName)

	sqlDB := openTenantSQLDB(*metaPath, tenantx.PublicID.String())
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Println(err)
		}
	}()

	rng := rand.New(rand.NewSource(*seed))

	start := time.Now()
	versionCount := fillSpaceWithFiles(sqlDB, spaceID, rootDirID, *count, *batchSize, *minVersions, *maxVersions, *prefix, rng)
	elapsed := time.Since(start)

	rate := float64(*count)
	if elapsed > 0 {
		rate = float64(*count) / elapsed.Seconds()
	}

	log.Printf("done: created space=%s (%s), files=%d, versions=%d, elapsed=%s, rate=%.2f files/s", spaceName, spacePublicID, *count, versionCount, elapsed.Round(time.Second), rate)
	log.Printf("browse URL path: /org/%s/space/%s/browse/", tenantx.PublicID.String(), spacePublicID)
}

func openMainDB(metaPath string) *sqlx.MainDB {
	mainDBPath, err := securejoin.SecureJoin(metaPath, "main.sqlite3")
	if err != nil {
		log.Fatalln(err)
	}

	return sqlx.NewMainDB(mainDBPath)
}

func resolveTenant(mainDB *sqlx.MainDB, tenantPublicID string) *tenant.Tenant {
	ctx := mainprivacy.DecisionContext(context.Background(), mainprivacy.Allow)

	if tenantPublicID != "" {
		tenantx, err := mainDB.ReadOnlyConn.Tenant.Query().
			Where(
				tenant.PublicID(entx.NewCIText(tenantPublicID)),
				tenant.InitializedAtNotNil(),
			).
			Only(ctx)
		if err != nil {
			log.Fatalln(err)
		}
		return tenantx
	}

	tenants, err := mainDB.ReadOnlyConn.Tenant.Query().
		Where(tenant.InitializedAtNotNil()).
		Order(tenant.ByID()).
		All(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	if len(tenants) == 0 {
		log.Fatalln("no initialized tenant found")
	}

	if len(tenants) > 1 {
		log.Printf("multiple tenants found, using first: %s (pass -tenant to select another)", tenants[0].PublicID.String())
	}

	return tenants[0]
}

func openTenantDB(tenantx *tenant.Tenant, metaPath string) *sqlx.TenantDB {
	tenantDB, err := tenantm.NewTenant(tenantx).OpenDB(false, metaPath)
	if err != nil {
		log.Fatalln(err)
	}

	return tenantDB
}

func confirmRun(confirmed bool, tenantPublicID string, fileCount int, estimatedVersions int64, prefix string) error {
	if confirmed {
		return nil
	}

	fmt.Printf("About to create a new space in tenant %q with %d files and about %d versions (prefix=%q).\n", tenantPublicID, fileCount, estimatedVersions, prefix)
	fmt.Print("Type 'yes' to continue: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("aborted")
	}

	if strings.TrimSpace(strings.ToLower(scanner.Text())) != "yes" {
		return fmt.Errorf("aborted")
	}

	return nil
}

func createSpaceAndRootDir(tenantDB *sqlx.TenantDB, spaceName string) (int64, string, int64) {
	ctx := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
	now := time.Now()

	tx, err := tenantDB.ReadWriteConn.Tx(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	spacex, err := tx.Space.Create().
		SetName(spaceName).
		SetIsFolderMode(true).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		log.Fatalln(err)
	}

	rootDirx, err := tx.File.Create().
		SetName(spaceName).
		SetIsDirectory(true).
		SetIndexedAt(now).
		SetModifiedAt(now).
		SetSpaceID(spacex.ID).
		SetIsRootDir(true).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		log.Fatalln(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalln(err)
	}

	log.Printf("created space %q (%s), root_dir_id=%d", spaceName, spacex.PublicID.String(), rootDirx.ID)

	return spacex.ID, spacex.PublicID.String(), rootDirx.ID
}

func openTenantSQLDB(metaPath, tenantPublicID string) *sql.DB {
	tenantDirPath, err := pathx.TenantDBPath(metaPath, tenantPublicID)
	if err != nil {
		log.Fatalln(err)
	}

	tenantDBPath, err := securejoin.SecureJoin(tenantDirPath, "tenant.sqlite3")
	if err != nil {
		log.Fatalln(err)
	}

	dataSourceURL := fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadWrite)
	sqlDB, err := sql.Open("sqlite3", dataSourceURL)
	if err != nil {
		log.Fatalln(err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		log.Fatalln(err)
	}

	return sqlDB
}

func fillSpaceWithFiles(
	sqlDB *sql.DB,
	spaceID int64,
	rootDirID int64,
	totalFileCount int,
	batchSize int,
	minVersions int,
	maxVersions int,
	prefix string,
	rng *rand.Rand,
) int64 {
	insertFileSQL := "INSERT INTO files (public_id, created_at, updated_at, space_id, name, is_directory, modified_at, indexed_at, parent_id, is_in_inbox, is_root_dir, ocr_content, ocr_retry_count, ocr_last_tried_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	insertStoredFileSQL := "INSERT INTO stored_files (created_at, updated_at, upload_started_at, upload_succeeded_at, filename, size, size_in_storage, mime_type, storage_type, storage_path, storage_filename, temporary_storage_path, temporary_storage_filename) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	insertVersionSQL := "INSERT INTO file_versions (file_id, stored_file_id, version_number) VALUES (?, ?, ?)"

	start := time.Now()
	insertedFileCount := 0
	insertedVersionCount := int64(0)

	for insertedFileCount < totalFileCount {
		remaining := totalFileCount - insertedFileCount
		currentBatchSize := batchSize
		if remaining < currentBatchSize {
			currentBatchSize = remaining
		}

		tx, err := sqlDB.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatalln(err)
		}

		fileStmt, err := tx.PrepareContext(context.Background(), insertFileSQL)
		if err != nil {
			_ = tx.Rollback()
			log.Fatalln(err)
		}

		storedFileStmt, err := tx.PrepareContext(context.Background(), insertStoredFileSQL)
		if err != nil {
			_ = fileStmt.Close()
			_ = tx.Rollback()
			log.Fatalln(err)
		}

		versionStmt, err := tx.PrepareContext(context.Background(), insertVersionSQL)
		if err != nil {
			_ = storedFileStmt.Close()
			_ = fileStmt.Close()
			_ = tx.Rollback()
			log.Fatalln(err)
		}

		now := time.Now().UTC()

		for i := 0; i < currentBatchSize; i++ {
			fileIndex := insertedFileCount + i
			filePublicID := util.NewPublicID()
			fileName := fmt.Sprintf("%s-file-%07d.pdf", prefix, fileIndex)

			fileResult, err := fileStmt.ExecContext(
				context.Background(),
				filePublicID,
				now,
				now,
				spaceID,
				fileName,
				false,
				now,
				now,
				rootDirID,
				false,
				false,
				"",
				0,
				time.Time{},
			)
			if err != nil {
				_ = versionStmt.Close()
				_ = storedFileStmt.Close()
				_ = fileStmt.Close()
				_ = tx.Rollback()
				log.Fatalln(err)
			}

			fileID, err := fileResult.LastInsertId()
			if err != nil {
				_ = versionStmt.Close()
				_ = storedFileStmt.Close()
				_ = fileStmt.Close()
				_ = tx.Rollback()
				log.Fatalln(err)
			}

			versionCount := minVersions + rng.Intn(maxVersions-minVersions+1)
			for versionNumber := 1; versionNumber <= versionCount; versionNumber++ {
				mimeType := "application/pdf"
				storageFilename := fmt.Sprintf("%s-v%d.pdf", filePublicID, versionNumber)
				if versionNumber == versionCount && fileIndex%10 == 0 {
					mimeType = "application/zip"
					storageFilename = fmt.Sprintf("%s-v%d.zip", filePublicID, versionNumber)
				}

				storedFileResult, err := storedFileStmt.ExecContext(
					context.Background(),
					now,
					now,
					now,
					now,
					fileName,
					int64(1024),
					int64(1024),
					mimeType,
					"Local",
					"benchmark/storage",
					storageFilename,
					"benchmark/tmp",
					fmt.Sprintf("tmp-%s-v%d", filePublicID, versionNumber),
				)
				if err != nil {
					_ = versionStmt.Close()
					_ = storedFileStmt.Close()
					_ = fileStmt.Close()
					_ = tx.Rollback()
					log.Fatalln(err)
				}

				storedFileID, err := storedFileResult.LastInsertId()
				if err != nil {
					_ = versionStmt.Close()
					_ = storedFileStmt.Close()
					_ = fileStmt.Close()
					_ = tx.Rollback()
					log.Fatalln(err)
				}

				_, err = versionStmt.ExecContext(context.Background(), fileID, storedFileID, versionNumber)
				if err != nil {
					_ = versionStmt.Close()
					_ = storedFileStmt.Close()
					_ = fileStmt.Close()
					_ = tx.Rollback()
					log.Fatalln(err)
				}

				insertedVersionCount++
			}
		}

		if err := versionStmt.Close(); err != nil {
			_ = storedFileStmt.Close()
			_ = fileStmt.Close()
			_ = tx.Rollback()
			log.Fatalln(err)
		}
		if err := storedFileStmt.Close(); err != nil {
			_ = fileStmt.Close()
			_ = tx.Rollback()
			log.Fatalln(err)
		}
		if err := fileStmt.Close(); err != nil {
			_ = tx.Rollback()
			log.Fatalln(err)
		}
		if err := tx.Commit(); err != nil {
			log.Fatalln(err)
		}

		insertedFileCount += currentBatchSize

		elapsed := time.Since(start)
		rate := float64(insertedFileCount)
		if elapsed > 0 {
			rate = float64(insertedFileCount) / elapsed.Seconds()
		}

		log.Printf("progress: files=%d/%d versions=%d elapsed=%s rate=%.2f files/s", insertedFileCount, totalFileCount, insertedVersionCount, elapsed.Round(time.Second), rate)
	}

	return insertedVersionCount
}
