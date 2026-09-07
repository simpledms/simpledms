package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

func TestDocumentNotesModelLifecycleAndPersistence(t *testing.T) {
	harness := newActionTestHarness(t)
	account, tenant := signUpAccount(t, harness, "notes-owner@example.com")
	tenantDB := initTenantDB(t, harness, tenant)
	tenant = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)
	notes := filemodel.NewDocumentNotes()
	var docID, predecessorID, successorID string
	run := func(fn func(*ctxx.SpaceContext) error) error {
		return withTenantContext(t, harness, account, tenant, tenantDB, func(
			_ *entmain.Tx, _ *enttenant.Tx, ctx *ctxx.TenantContext,
		) error {
			spacex, err := ctx.TTx.Space.Query().Where(space.Name("Notes")).Only(ctx)
			if enttenant.IsNotFound(err) {
				createSpaceViaCmd(t, harness.actions, ctx, "Notes")
				spacex = ctx.TTx.Space.Query().Where(space.Name("Notes")).OnlyX(ctx)
			} else if err != nil {
				return err
			}
			return fn(ctxx.NewSpaceContext(ctx, spacex))
		})
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		doc := createDocumentForNotesTest(ctx, "notes.pdf", "legacy\n  text")
		docID = doc.PublicID.String()
		loaded, rows, err := notes.List(ctx, docID, false)
		if err != nil || len(rows) != 0 || loaded.Notes != "legacy\n  text" {
			t.Fatalf("read legacy without materialization: %v, %v", rows, err)
		}
		if _, err := notes.Create(ctx, docID, "Initial title", " \n\t"); err == nil {
			t.Fatal("accepted empty text")
		}
		first, err := notes.Create(ctx, docID, "Initial title", "first\n<script>plain text</script>")
		if err != nil {
			return err
		}
		second, err := notes.Create(ctx, docID, "Second title", "second")
		if err != nil {
			return err
		}
		// Equal authored timestamps still have a stable newest-first ID tie-breaker.
		ctx.TTx.DocumentNote.UpdateOne(second).SetAuthoredAt(*first.AuthoredAt).SaveX(ctx)
		_, rows, err = notes.List(ctx, docID, false)
		if err != nil || len(rows) != 2 || rows[0].ID != second.ID ||
			rows[1].AuthorID != ctx.User.ID || rows[1].Edges.Author == nil ||
			rows[1].Title != "Initial title" || rows[0].Title != "Second title" {
			t.Fatalf("order/authorship: %v, %v", rows, err)
		}
		edited, err := notes.Edit(
			ctx,
			docID,
			first.PublicID.String(),
			"Corrected title",
			"corrected\ntext",
		)
		if err != nil || edited.ID != first.ID || edited.AuthorID != first.AuthorID ||
			!edited.AuthoredAt.Equal(*first.AuthoredAt) || edited.EditedAt == nil ||
			edited.EditorID != ctx.User.ID || edited.Title != "Corrected title" {
			t.Fatalf("edit attribution: %v, %v", edited, err)
		}
		legacy, err := notes.Edit(ctx, docID, "legacy", "Legacy title", "corrected legacy")
		if err != nil || legacy.AuthorID != 0 || legacy.AuthoredAt != nil {
			t.Fatalf("legacy attribution: %v, %v", legacy, err)
		}
		loaded, rows, err = notes.List(ctx, docID, true)
		if err != nil || loaded.Notes != "" || len(rows) != 3 || rows[2].ID != legacy.ID {
			t.Fatalf("legacy representation/date ordering: %v, %v", rows, err)
		}
		if _, _, err := notes.Get(ctx, docID, "legacy"); err == nil {
			t.Fatal("materialized legacy selector remained valid")
		}
		predecessorID = first.PublicID.String()
		replacement, err := notes.Replace(ctx, docID, predecessorID, "Replacement title", "replacement")
		if err != nil {
			return err
		}
		successor, err := notes.Replace(
			ctx,
			docID,
			replacement.PublicID.String(),
			"Successor title",
			"successor",
		)
		if err != nil {
			return err
		}
		successorID = successor.PublicID.String()
		if _, err := notes.Replace(ctx, docID, predecessorID, "Stale title", "stale"); err == nil {
			t.Fatal("accepted competing replacement")
		}
		if _, err := notes.Edit(ctx, docID, predecessorID, "Stale title", "stale"); err == nil {
			t.Fatal("edited replaced note")
		}
		if _, err := notes.Delete(ctx, docID, predecessorID); err == nil {
			t.Fatal("deleted replaced note")
		}
		if _, err := notes.Delete(ctx, docID, successorID); err != nil {
			return err
		}
		if _, err := notes.Delete(ctx, docID, successorID); err == nil {
			t.Fatal("accepted stale deletion")
		}
		_, rows, err = notes.List(ctx, docID, false)
		if err != nil || len(rows) != 2 {
			t.Fatalf("predecessors reactivated: %v, %v", rows, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		doc, rows, err := notes.List(ctx, docID, true)
		if err != nil || len(rows) != 5 || doc.Notes != "" {
			t.Fatalf("persisted history: %v, %v", rows, err)
		}
		_, old, err := notes.Get(ctx, docID, predecessorID)
		if err != nil || old.Body != "corrected\ntext" || old.Edges.Replacement == nil ||
			old.Title != "Corrected title" || old.Edges.Replacement.Title != "Replacement title" {
			t.Fatalf("retained predecessor: %v, %v", old, err)
		}
		ctx.TTx.File.UpdateOneID(doc.ID).SetDeletedAt(time.Now()).SaveX(ctx)
		trashed, _, err := notes.List(ctx, docID, true)
		if err != nil || trashed.DeletedAt.IsZero() || notes.CanChange(ctx, doc, ctx.User.ID) {
			t.Fatalf("Trash reads/change guard: %v", err)
		}
		if _, err := notes.Create(ctx, docID, "Trash title", "trash"); err == nil {
			t.Fatal("created in Trash")
		}
		for _, note := range rows {
			if _, err := notes.Edit(ctx, docID, note.PublicID.String(), "Trash title", "trash"); err == nil {
				t.Fatal("edited in Trash")
			}
			if _, err := notes.Replace(
				ctx,
				docID,
				note.PublicID.String(),
				"Trash title",
				"trash",
			); err == nil {
				t.Fatal("replaced in Trash")
			}
			if _, err := notes.Delete(ctx, docID, note.PublicID.String()); err == nil {
				t.Fatal("deleted in Trash")
			}
		}
		ctx.TTx.File.UpdateOneID(doc.ID).ClearDeletedAt().SaveX(schema.SkipSoftDelete(ctx))
		_, rows, err = notes.List(ctx, docID, false)
		if err != nil || len(rows) != 2 {
			t.Fatalf("restore changed history: %v, %v", rows, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesModelPermissionsAndScope(t *testing.T) {
	harness := newActionTestHarness(t)
	account, tenant := signUpAccount(t, harness, "notes-permissions@example.com")
	tenantDB := initTenantDB(t, harness, tenant)
	tenant = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)
	err := withTenantContext(t, harness, account, tenant, tenantDB, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Permissions")
		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Permissions")).OnlyX(tenantCtx)
		ctx := ctxx.NewSpaceContext(tenantCtx, spacex)
		notes := filemodel.NewDocumentNotes()
		doc := createDocumentForNotesTest(ctx, "first.pdf", "legacy")
		other := createDocumentForNotesTest(ctx, "other.pdf", "")
		note, err := notes.Create(ctx, doc.PublicID.String(), "Actor title", "actor note")
		if err != nil {
			return err
		}
		if _, _, err := notes.Get(ctx, other.PublicID.String(), note.PublicID.String()); err == nil {
			t.Fatal("accepted mismatched document/note")
		}
		if _, _, err := notes.List(ctx, ctx.SpaceRootDir().PublicID.String(), true); err == nil {
			t.Fatal("accepted directory")
		}
		if _, _, err := notes.List(ctx, "unknown-document", true); err == nil {
			t.Fatal("accepted unknown document")
		}
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Other Space")
		otherSpace := ctx.TTx.Space.Query().Where(space.Name("Other Space")).OnlyX(ctx)
		otherCtx := ctxx.NewSpaceContext(tenantCtx, otherSpace)
		foreign := createDocumentForNotesTest(otherCtx, "foreign.pdf", "")
		if _, _, err := notes.List(otherCtx, doc.PublicID.String(), true); err == nil {
			t.Fatal("cross-Space read")
		}
		if _, err := notes.Transfer(ctx, doc, foreign); err == nil {
			t.Fatal("cross-Space transfer")
		}
		otherAuthor := ctx.TTx.User.Create().SetAccountID(account.ID + 10000).
			SetEmail(entx.NewCIText("other-author@example.com")).SetFirstName("Other").
			SetRole(tenantrole.User).SaveX(ctx)
		otherNote := ctx.TTx.DocumentNote.Create().SetSpaceID(spacex.ID).SetFileID(doc.ID).
			SetBody("another author").SetAuthorID(otherAuthor.ID).SetAuthoredAt(time.Now()).SaveX(ctx)
		// Tenant ownership is read from storage, not the old context's cached role.
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		assignment := ctx.TTx.SpaceUserAssignment.Query().Where(
			spaceuserassignment.SpaceID(spacex.ID), spaceuserassignment.UserID(ctx.User.ID),
		).OnlyX(ctx)
		ctx.TTx.SpaceUserAssignment.UpdateOne(assignment).SetRole(spacerole.User).SaveX(ctx)
		if !notes.CanChange(ctx, doc, note.AuthorID) || notes.CanChange(ctx, doc, otherAuthor.ID) ||
			notes.CanChange(ctx, doc, 0) {
			t.Fatal("member permission matrix")
		}
		if _, err := notes.Edit(
			ctx,
			doc.PublicID.String(),
			otherNote.PublicID.String(),
			"Forbidden title",
			"forbidden",
		); err == nil {
			t.Fatal("member edited another author's note")
		}
		if _, err := notes.Replace(
			ctx,
			doc.PublicID.String(),
			otherNote.PublicID.String(),
			"Forbidden title",
			"forbidden",
		); err == nil {
			t.Fatal("member replaced another author's note")
		}
		if _, err := notes.Delete(ctx, doc.PublicID.String(), otherNote.PublicID.String()); err == nil {
			t.Fatal("member deleted another author's note")
		}
		if _, err := notes.Edit(
			ctx,
			doc.PublicID.String(),
			"legacy",
			"Forbidden",
			"forbidden",
		); err == nil {
			t.Fatal("member edited unknown author")
		}
		ctx.TTx.SpaceUserAssignment.UpdateOne(assignment).SetRole(spacerole.Owner).SaveX(ctx)
		edited, err := notes.Edit(
			ctx,
			doc.PublicID.String(),
			otherNote.PublicID.String(),
			"Owner",
			"owner edit",
		)
		if err != nil || edited.AuthorID != otherAuthor.ID || edited.EditorID != ctx.User.ID {
			t.Fatalf("Space owner attribution: %v, %v", edited, err)
		}
		ctx.TTx.SpaceUserAssignment.DeleteOne(assignment).ExecX(ctx)
		if _, _, err := notes.List(ctx, doc.PublicID.String(), true); err == nil {
			t.Fatal("revoked author retained read access")
		}
		if _, err := notes.Edit(
			ctx,
			doc.PublicID.String(),
			note.PublicID.String(),
			"Revoked title",
			"revoked",
		); err == nil {
			t.Fatal("revoked author retained edit access")
		}
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.Owner).SaveX(ctx)
		if !notes.CanChange(ctx, doc, otherAuthor.ID) || !notes.CanChange(ctx, doc, 0) {
			t.Fatal("tenant owner requires Space assignment")
		}
		ctx.MainTx.TenantAccountAssignment.Update().Where(
			tenantaccountassignment.TenantID(tenant.ID),
			tenantaccountassignment.AccountID(account.ID),
		).SetExpiresAt(time.Now().Add(-time.Hour)).SaveX(ctx)
		if _, _, err := notes.List(ctx, doc.PublicID.String(), true); err == nil {
			t.Fatal("expired tenant membership retained access")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesModelTransferAndRollback(t *testing.T) {
	harness := newActionTestHarness(t)
	account, tenant := signUpAccount(t, harness, "notes-transfer@example.com")
	tenantDB := initTenantDB(t, harness, tenant)
	tenant = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)
	notes := filemodel.NewDocumentNotes()
	var sourceID, targetID, currentID string
	run := func(fn func(*ctxx.SpaceContext) error) error {
		return withTenantContext(t, harness, account, tenant, tenantDB, func(
			_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
		) error {
			spacex, err := tenantCtx.TTx.Space.Query().Where(space.Name("Transfer")).Only(tenantCtx)
			if enttenant.IsNotFound(err) {
				createSpaceViaCmd(t, harness.actions, tenantCtx, "Transfer")
				spacex = tenantCtx.TTx.Space.Query().Where(space.Name("Transfer")).OnlyX(tenantCtx)
			} else if err != nil {
				return err
			}
			return fn(ctxx.NewSpaceContext(tenantCtx, spacex))
		})
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		source := createDocumentForNotesTest(ctx, "source.pdf", "source legacy")
		target := createDocumentForNotesTest(ctx, "target.pdf", "target legacy")
		sourceID, targetID = source.PublicID.String(), target.PublicID.String()
		first, err := notes.Create(ctx, sourceID, "Original title", "old")
		if err != nil {
			return err
		}
		next, err := notes.Replace(ctx, sourceID, first.PublicID.String(), "Next title", "next")
		if err != nil {
			return err
		}
		if _, err := notes.Edit(
			ctx,
			sourceID,
			next.PublicID.String(),
			"Edited title",
			"edited next",
		); err != nil {
			return err
		}
		currentID = next.PublicID.String()
		deleted, err := notes.Create(ctx, sourceID, "Deleted title", "deleted")
		if err != nil {
			return err
		}
		if _, err := notes.Delete(ctx, sourceID, deleted.PublicID.String()); err != nil {
			return err
		}
		_, err = notes.Create(ctx, targetID, "Target title", "target note")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("abort owning transaction")
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if _, err := notes.Replace(
			ctx,
			sourceID,
			currentID,
			"Rolled back",
			"rolled back successor",
		); err != nil {
			return err
		}
		source, _, err := notes.List(ctx, sourceID, true)
		if err != nil {
			return err
		}
		target, _, err := notes.List(ctx, targetID, true)
		if err != nil {
			return err
		}
		if _, err := notes.Transfer(ctx, source, target); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("rollback: %v", err)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		source, sourceNotes, err := notes.List(ctx, sourceID, true)
		if err != nil || len(sourceNotes) != 3 || source.Notes != "source legacy" {
			t.Fatalf("source rollback: %v, %v", sourceNotes, err)
		}
		target, targetNotes, err := notes.List(ctx, targetID, true)
		if err != nil || len(targetNotes) != 1 || target.Notes != "target legacy" {
			t.Fatalf("target rollback: %v, %v", targetNotes, err)
		}
		_, current, err := notes.Get(ctx, sourceID, currentID)
		if err != nil || current.ReplacedByID != 0 || current.Body != "edited next" ||
			current.Title != "Edited title" {
			t.Fatalf("replacement rollback: %v, %v", current, err)
		}
		if _, err := notes.Transfer(ctx, source, target); err != nil {
			return err
		}
		// A repeated transfer with the old source object must re-read cleared legacy text.
		if _, err := notes.Transfer(ctx, source, target); err != nil {
			return err
		}
		ctx.TTx.File.DeleteOneID(source.ID).ExecX(schema.SkipSoftDelete(ctx))
		_, rows, err := notes.List(ctx, targetID, true)
		if err != nil || len(rows) != 5 {
			t.Fatalf("transfer all history without duplicates: %v, %v", rows, err)
		}
		for _, before := range sourceNotes {
			after := ctx.TTx.DocumentNote.Query().Where(documentnote.ID(before.ID)).OnlyX(ctx)
			if after.FileID != target.ID || after.AuthorID != before.AuthorID ||
				after.Body != before.Body || after.ReplacedByID != before.ReplacedByID ||
				after.EditorID != before.EditorID || after.Title != before.Title {
				t.Fatalf("transfer changed note attribution/history: %v -> %v", before, after)
			}
		}
		if _, err := notes.Transfer(ctx, source, target); err == nil {
			t.Fatal("accepted already deleted source")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesModelInboxMergePreservesHistory(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)
	account, tenant := signUpAccount(t, harness, "notes-merge@example.com")
	tenantDB := initTenantDB(t, harness, tenant)
	tenant = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)
	err := withTenantContext(t, harness, account, tenant, tenantDB, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Merge notes")
		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Merge notes")).OnlyX(tenantCtx)
		ctx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootID := ctx.SpaceRootDir().ID
		source := uploadSpaceFile(t, harness, ctx, rootID, "source.pdf", []byte("source"), true)
		target := uploadSpaceFile(t, harness, ctx, rootID, "target.pdf", []byte("target"), false)
		source.Update().SetNotes("legacy from Inbox").SaveX(ctx)
		notes := filemodel.NewDocumentNotes()
		original, err := notes.Create(ctx, source.PublicID.String(), "Original title", "original")
		if err != nil {
			return err
		}
		replacement, err := notes.Replace(
			ctx,
			source.PublicID.String(),
			original.PublicID.String(),
			"New title",
			"new",
		)
		if err != nil {
			return err
		}
		if _, err := notes.Delete(
			ctx,
			source.PublicID.String(),
			replacement.PublicID.String(),
		); err != nil {
			return err
		}
		if _, err := notes.Create(
			ctx,
			target.PublicID.String(),
			"Target title",
			"target note",
		); err != nil {
			return err
		}
		if _, err := runFileVersionFromInboxCmd(
			harness, ctx, target.PublicID.String(), source.PublicID.String(),
		); err != nil {
			return err
		}
		_, rows, err := notes.List(ctx, target.PublicID.String(), true)
		if err != nil || len(rows) != 4 {
			t.Fatalf("merged history: %v, %v", rows, err)
		}
		_, old, err := notes.Get(ctx, target.PublicID.String(), original.PublicID.String())
		if err != nil || old.Edges.Replacement == nil || old.ReplacedByID != replacement.ID ||
			old.AuthorID != ctx.User.ID || old.Edges.Replacement.DeletedAt == nil ||
			old.Title != "Original title" || old.Edges.Replacement.Title != "New title" {
			t.Fatalf("merged relationship/state: %v, %v", old, err)
		}
		if _, _, err := notes.List(ctx, source.PublicID.String(), true); err == nil {
			t.Fatal("merged source still exists")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func createDocumentForNotesTest(
	ctx *ctxx.SpaceContext, name, legacy string,
) *enttenant.File {
	return ctx.TTx.File.Create().SetSpaceID(ctx.Space.ID).SetName(name).
		SetIsDirectory(false).SetIndexedAt(time.Now()).SetParentID(ctx.SpaceRootDir().ID).
		SetNotes(legacy).SaveX(ctx)
}
