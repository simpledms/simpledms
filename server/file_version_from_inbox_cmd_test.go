package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"entgo.io/ent/dialect/sql"

	"github.com/marcobeierer/go-core/db/entmain"

	"github.com/marcobeierer/go-core/ctxx"
	ctxx2 "github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
)

func TestFileVersionFromInboxCmd_MergesVersionAndDeletesSource(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
			spaceName := "Inbox Merge Space"
			createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
			rootDir := spaceCtx.SpaceRootDir()

			targetFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"target.pdf",
				[]byte("target content"),
				false,
			)
			sourceFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"source.pdf",
				[]byte("source content"),
				true,
			)

			targetVersionsBefore := spaceCtx.TTx.FileVersion.Query().Where(fileversion.FileID(targetFile.ID)).CountX(spaceCtx)
			sourceLatestVersionBefore := latestFileVersion(spaceCtx, sourceFile.ID)

			rr, err := runFileVersionFromInboxCmd(
				harness,
				spaceCtx,
				targetFile.PublicID.String(),
				sourceFile.PublicID.String(),
			)
			if err != nil {
				return fmt.Errorf("file version from inbox command: %w", err)
			}

			expectedTrigger := fmt.Sprintf(
				"%s, %s, %s, %s",
				event.FileUploaded.String(),
				event.FileUpdated.String(),
				event.FileDeleted.String(),
				events.CloseDialog.String(),
			)
			if got := rr.Header().Get("HX-Trigger"); got != expectedTrigger {
				return fmt.Errorf("expected HX-Trigger %q, got %q", expectedTrigger, got)
			}

			targetVersionsAfter := spaceCtx.TTx.FileVersion.Query().Where(fileversion.FileID(targetFile.ID)).CountX(spaceCtx)
			if targetVersionsAfter != targetVersionsBefore+1 {
				return fmt.Errorf("expected target versions %d, got %d", targetVersionsBefore+1, targetVersionsAfter)
			}

			targetLatestVersionAfter := latestFileVersion(spaceCtx, targetFile.ID)
			if targetLatestVersionAfter.StoredFileID != sourceLatestVersionBefore.StoredFileID {
				return fmt.Errorf(
					"expected merged stored file id %d, got %d",
					sourceLatestVersionBefore.StoredFileID,
					targetLatestVersionAfter.StoredFileID,
				)
			}

			targetAfter := spaceCtx.TTx.File.Query().Where(file.ID(targetFile.ID)).OnlyX(spaceCtx)
			if targetAfter.Name != sourceFile.Name {
				return fmt.Errorf("expected target name %q, got %q", sourceFile.Name, targetAfter.Name)
			}

			sourceExistsWithDeleted := spaceCtx.TTx.File.Query().Where(file.ID(sourceFile.ID)).ExistX(schema.SkipSoftDelete(spaceCtx))
			if sourceExistsWithDeleted {
				return fmt.Errorf("expected source file to be hard deleted")
			}

			sourceVersionCount := spaceCtx.TTx.FileVersion.Query().Where(fileversion.FileID(sourceFile.ID)).CountX(spaceCtx)
			if sourceVersionCount != 0 {
				return fmt.Errorf("expected source versions to be deleted, got %d", sourceVersionCount)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("file version from inbox command: %v", err)
		}
	})
}

func TestFileVersionFromInboxCmd_RejectsSameSourceAndTarget(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		var handlerErr error
		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
			spaceName := "Inbox Merge Same File Space"
			createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
			rootDir := spaceCtx.SpaceRootDir()

			sourceFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"source.pdf",
				[]byte("source content"),
				true,
			)

			_, handlerErr = runFileVersionFromInboxCmd(
				harness,
				spaceCtx,
				sourceFile.PublicID.String(),
				sourceFile.PublicID.String(),
			)
			if handlerErr == nil {
				return fmt.Errorf("expected error")
			}

			return nil
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, httpErr.StatusCode())
		}
		if !strings.Contains(httpErr.Error(), "source and target must be different files") {
			t.Fatalf("expected source/target error, got %q", httpErr.Error())
		}
	})
}

func TestFileVersionFromInboxCmd_RejectsSourceOutsideInbox(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		var handlerErr error
		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
			spaceName := "Inbox Merge Outside Inbox Space"
			createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
			rootDir := spaceCtx.SpaceRootDir()

			targetFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"target.pdf",
				[]byte("target content"),
				false,
			)
			sourceFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"source.pdf",
				[]byte("source content"),
				false,
			)

			_, handlerErr = runFileVersionFromInboxCmd(
				harness,
				spaceCtx,
				targetFile.PublicID.String(),
				sourceFile.PublicID.String(),
			)
			if handlerErr == nil {
				return fmt.Errorf("expected error")
			}

			return nil
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, httpErr.StatusCode())
		}
		if !strings.Contains(httpErr.Error(), "file must be in inbox") {
			t.Fatalf("expected inbox validation error, got %q", httpErr.Error())
		}
	})
}

func TestFileVersionFromInboxCmd_RejectsSourceWithoutVersion(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		var handlerErr error
		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
			spaceName := "Inbox Merge No Version Space"
			createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
			rootDir := spaceCtx.SpaceRootDir()

			targetFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"target.pdf",
				[]byte("target content"),
				false,
			)
			sourceFile := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				rootDir.ID,
				"source.pdf",
				[]byte("source content"),
				true,
			)

			_, err := spaceCtx.TTx.FileVersion.Delete().Where(fileversion.FileID(sourceFile.ID)).Exec(spaceCtx)
			if err != nil {
				return fmt.Errorf("delete source versions: %w", err)
			}

			targetVersionsBefore := spaceCtx.TTx.FileVersion.Query().Where(fileversion.FileID(targetFile.ID)).CountX(spaceCtx)

			_, handlerErr = runFileVersionFromInboxCmd(
				harness,
				spaceCtx,
				targetFile.PublicID.String(),
				sourceFile.PublicID.String(),
			)
			if handlerErr == nil {
				return fmt.Errorf("expected error")
			}

			targetVersionsAfter := spaceCtx.TTx.FileVersion.Query().Where(fileversion.FileID(targetFile.ID)).CountX(spaceCtx)
			if targetVersionsAfter != targetVersionsBefore {
				return fmt.Errorf("expected target versions to remain %d, got %d", targetVersionsBefore, targetVersionsAfter)
			}

			sourceExists := spaceCtx.TTx.File.Query().Where(file.ID(sourceFile.ID)).ExistX(schema.SkipSoftDelete(spaceCtx))
			if !sourceExists {
				return fmt.Errorf("expected source file to remain when merge fails")
			}

			return nil
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, httpErr.StatusCode())
		}
		if !strings.Contains(httpErr.Error(), "source file has no versions") {
			t.Fatalf("expected no-version error, got %q", httpErr.Error())
		}
	})
}

func runFileVersionFromInboxCmd(
	harness *actionTestHarness,
	spaceCtx *ctxx.SpaceContext,
	targetFileID string,
	sourceFileID string,
) (*httptest.ResponseRecorder, error) {
	form := url.Values{}
	form.Set("TargetFileID", targetFileID)
	form.Set("SourceFileID", sourceFileID)
	form.Set("ConfirmWarning", "true")

	req := httptest.NewRequest(http.MethodPost, "/-/browse/file-version-from-inbox-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	err := harness.actions.Browse.FileVersionFromInboxCmd.Handler(
		httpx2.NewResponseWriter(rr),
		httpx2.NewRequest(req),
		spaceCtx,
	)

	return rr, err
}

func uploadSpaceFile(
	t testing.TB,
	harness *actionTestHarness,
	spaceCtx *ctxx.SpaceContext,
	parentDirID int64,
	filename string,
	content []byte,
	isInInbox bool,
) *enttenant.File {
	t.Helper()

	prepared, filex, err := harness.infra.FileSystem().PrepareFileUpload(
		spaceCtx,
		filename,
		parentDirID,
		isInInbox,
	)
	if err != nil {
		t.Fatalf("prepare file upload: %v", err)
	}

	fileInfo, fileSize, err := harness.infra.FileSystem().UploadPreparedFileWithExpectedSize(
		spaceCtx,
		bytes.NewReader(content),
		prepared,
		int64(len(content)),
	)
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}

	err = harness.infra.FileSystem().FinalizePreparedUpload(spaceCtx, prepared, fileInfo, fileSize)
	if err != nil {
		t.Fatalf("finalize file upload: %v", err)
	}

	return filex
}

func latestFileVersion(spaceCtx *ctxx.SpaceContext, fileID int64) *enttenant.FileVersion {
	return spaceCtx.TTx.FileVersion.Query().
		Where(fileversion.FileID(fileID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		FirstX(spaceCtx)
}
