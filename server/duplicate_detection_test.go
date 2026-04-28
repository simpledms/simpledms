package server

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

func TestDuplicateDetectionFindsAccessibleSpaceAndHistoricVersion(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx := signUpAccount(t, harness, "duplicate-owner@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			createSpaceViaCmd(t, harness.actions, tenantCtx, "Duplicate Source Space")
			createSpaceViaCmd(t, harness.actions, tenantCtx, "Duplicate Other Space")

			sourceSpace := tenantCtx.TTx.Space.Query().Where(space.Name("Duplicate Source Space")).OnlyX(tenantCtx)
			otherSpace := tenantCtx.TTx.Space.Query().Where(space.Name("Duplicate Other Space")).OnlyX(tenantCtx)
			sourceSpaceCtx := ctxx.NewSpaceContext(tenantCtx, sourceSpace)

			content := []byte("same duplicate content")
			sourceFile := uploadSpaceFile(
				t,
				harness,
				sourceSpaceCtx,
				sourceSpaceCtx.SpaceRootDir().ID,
				"inbox-source.pdf",
				content,
				true,
			)
			uploadFileVersion(t, harness, sourceSpaceCtx, sourceFile.ID, "inbox-source-v2.pdf", content)
			historicFile := uploadSpaceFile(
				t,
				harness,
				sourceSpaceCtx,
				sourceSpaceCtx.SpaceRootDir().ID,
				"historic.pdf",
				content,
				false,
			)
			deletedFile := uploadSpaceFile(
				t,
				harness,
				sourceSpaceCtx,
				sourceSpaceCtx.SpaceRootDir().ID,
				"deleted.pdf",
				content,
				false,
			)
			sourceSpaceCtx.TTx.File.UpdateOne(deletedFile).
				SetDeletedAt(time.Now()).
				SaveX(sourceSpaceCtx)
			uploadFileVersion(t, harness, sourceSpaceCtx, historicFile.ID, "historic-new.pdf", []byte("new content"))

			otherSpaceCtx := ctxx.NewSpaceContext(tenantCtx, otherSpace)
			uploadSpaceFile(
				t,
				harness,
				otherSpaceCtx,
				otherSpaceCtx.SpaceRootDir().ID,
				"other-space.pdf",
				content,
				false,
			)

			sourceSpaceCtx = ctxx.NewSpaceContext(tenantCtx, sourceSpace)

			result, err := filemodel.NewDuplicateDetectionService().FindDuplicates(
				sourceSpaceCtx,
				sourceFile.PublicID.String(),
			)
			if err != nil {
				return err
			}
			if !result.HasContentHash {
				return fmt.Errorf("expected source content hash")
			}

			var foundOtherSpace bool
			var foundHistoricVersion bool
			for _, match := range result.Matches {
				if match.ParentDirPublicID == "" {
					return fmt.Errorf("expected duplicate match parent directory public ID")
				}
				if match.FilePublicID == sourceFile.PublicID.String() {
					return fmt.Errorf("expected source file versions to be excluded")
				}
				if match.FilePublicID == deletedFile.PublicID.String() {
					return fmt.Errorf("expected deleted files to be excluded")
				}
				if match.SpacePublicID == otherSpace.PublicID.String() && match.FileName == "other-space.pdf" {
					foundOtherSpace = true
				}
				if match.FilePublicID == historicFile.PublicID.String() && !match.IsCurrentVersion {
					foundHistoricVersion = true
				}
			}
			if !foundOtherSpace {
				return fmt.Errorf("expected duplicate in accessible other space")
			}
			if !foundHistoricVersion {
				return fmt.Errorf("expected duplicate in historic version")
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestDuplicateDetectionDoesNotLeakInaccessibleSpaces(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		ownerAccount, tenantx := signUpAccount(t, harness, "duplicate-owner-private@example.com")
		memberAccount := createTenantUser(
			t,
			harness,
			tenantx,
			"duplicate-member-private@example.com",
			tenantrole.User,
		)
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		var sourceSpacePublicID string
		var sourceFilePublicID string
		var privateFilePublicID string
		err := withTenantContext(t, harness, ownerAccount, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			memberUser := ensureTenantUserForAccount(t, tenantCtx, memberAccount, tenantrole.User)

			createSpaceViaCmd(t, harness.actions, tenantCtx, "Duplicate Member Source Space")
			createSpaceViaCmd(t, harness.actions, tenantCtx, "Duplicate Private Space")

			sourceSpace := tenantCtx.TTx.Space.Query().Where(space.Name("Duplicate Member Source Space")).OnlyX(tenantCtx)
			privateSpace := tenantCtx.TTx.Space.Query().Where(space.Name("Duplicate Private Space")).OnlyX(tenantCtx)

			err := tenantCtx.TTx.SpaceUserAssignment.Create().
				SetSpaceID(sourceSpace.ID).
				SetUserID(memberUser.ID).
				SetRole(spacerole.User).
				Exec(tenantCtx)
			if err != nil {
				return fmt.Errorf("assign member to source space: %w", err)
			}

			content := []byte("private duplicate content")
			sourceSpaceCtx := ctxx.NewSpaceContext(tenantCtx, sourceSpace)
			sourceFile := uploadSpaceFile(
				t,
				harness,
				sourceSpaceCtx,
				sourceSpaceCtx.SpaceRootDir().ID,
				"member-source.pdf",
				content,
				false,
			)

			privateSpaceCtx := ctxx.NewSpaceContext(tenantCtx, privateSpace)
			privateFile := uploadSpaceFile(
				t,
				harness,
				privateSpaceCtx,
				privateSpaceCtx.SpaceRootDir().ID,
				"private-match.pdf",
				content,
				false,
			)

			sourceSpacePublicID = sourceSpace.PublicID.String()
			sourceFilePublicID = sourceFile.PublicID.String()
			privateFilePublicID = privateFile.PublicID.String()
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		err = withTenantContext(t, harness, memberAccount, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			sourceSpace := tenantCtx.TTx.Space.Query().
				Where(space.PublicID(entx.NewCIText(sourceSpacePublicID))).
				OnlyX(tenantCtx)
			sourceSpaceCtx := ctxx.NewSpaceContext(tenantCtx, sourceSpace)

			result, err := filemodel.NewDuplicateDetectionService().FindDuplicates(
				sourceSpaceCtx,
				sourceFilePublicID,
			)
			if err != nil {
				return err
			}

			for _, match := range result.Matches {
				if match.FilePublicID == privateFilePublicID {
					return fmt.Errorf("expected inaccessible space duplicate to be excluded")
				}
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func uploadFileVersion(
	t testing.TB,
	harness *actionTestHarness,
	spaceCtx *ctxx.SpaceContext,
	fileID int64,
	filename string,
	content []byte,
) {
	t.Helper()

	prepared, err := harness.infra.FileSystem().PrepareFileVersionUpload(spaceCtx, filename, fileID)
	if err != nil {
		t.Fatalf("prepare file version upload: %v", err)
	}

	uploadResult, err := harness.infra.FileSystem().UploadPreparedFileWithExpectedSize(
		spaceCtx,
		bytes.NewReader(content),
		prepared,
		int64(len(content)),
	)
	if err != nil {
		t.Fatalf("upload file version: %v", err)
	}

	err = harness.infra.FileSystem().FinalizePreparedUpload(
		spaceCtx,
		prepared,
		uploadResult,
	)
	if err != nil {
		t.Fatalf("finalize file version upload: %v", err)
	}

	latestVersion := spaceCtx.TTx.FileVersion.Query().
		Where(fileversion.FileID(fileID)).
		Order(fileversion.ByVersionNumber()).
		AllX(spaceCtx)
	if len(latestVersion) < 2 {
		t.Fatalf("expected uploaded file version")
	}
}
