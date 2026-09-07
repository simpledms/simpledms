package file

import (
	"log"
	"net/http"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/util/e"
)

// DocumentNotes owns document note authorization and lifecycle transitions.
// Mutations must run in the caller's transaction, which must roll back on error.
type DocumentNotes struct{}

func NewDocumentNotes() *DocumentNotes {
	return &DocumentNotes{}
}

// List reads notes without materializing the separate, undated File.Notes legacy entry.
func (qq *DocumentNotes) List(
	ctx ctxx.Context, fileID string, showHistory bool,
) (*enttenant.File, []*enttenant.DocumentNote, error) {
	doc, err := qq.document(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	query := ctx.TenantCtx().TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).
		WithAuthor().WithEditor().WithReplacement()
	if !showHistory {
		query.Where(documentnote.DeletedAtIsNil(), documentnote.ReplacedByIDIsNil())
	}
	notes, err := query.Order(
		documentnote.ByAuthoredAt(sql.OrderDesc(), sql.OrderNullsLast()),
		documentnote.ByID(sql.OrderDesc()),
	).All(ctx)
	if err != nil {
		log.Printf("read document notes: %v", err)
		return nil, nil, err
	}
	return doc, notes, nil
}

// Get validates the document/note pair, including historical entries.
// The explicit "legacy" selector returns a nil note only while File.Notes is nonempty.
func (qq *DocumentNotes) Get(
	ctx ctxx.Context, fileID, noteID string,
) (*enttenant.File, *enttenant.DocumentNote, error) {
	doc, err := qq.document(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if noteID == "legacy" && doc.Notes != "" {
		return doc, nil, nil
	}
	note, err := ctx.TenantCtx().TTx.DocumentNote.Query().Where(
		documentnote.FileID(doc.ID), documentnote.PublicID(entx.NewCIText(noteID)),
	).WithAuthor().WithEditor().WithReplacement().Only(ctx)
	if err != nil {
		log.Printf("read document note: %v", err)
		return nil, nil, qq.failure(http.StatusNotFound, "Note not found.")
	}
	return doc, note, nil
}

// CanChange requires current document access and authorship or relevant ownership.
func (qq *DocumentNotes) CanChange(
	ctx ctxx.Context, doc *enttenant.File, authorID int64,
) bool {
	if doc == nil {
		return false
	}
	current, err := qq.document(ctx, doc.PublicID.String())
	if err != nil || current.ID != doc.ID || !current.DeletedAt.IsZero() {
		return false
	}
	isOwner, err := qq.access(ctx)
	return err == nil && (isOwner || authorID != 0 && authorID == ctx.TenantCtx().User.ID)
}

// Create adds a current note attributed exclusively to the authenticated actor.
func (qq *DocumentNotes) Create(
	ctx ctxx.Context, fileID, title, body string,
) (*enttenant.DocumentNote, error) {
	doc, err := qq.document(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if _, err := qq.editable(ctx, doc); err != nil {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		return nil, qq.failure(http.StatusBadRequest, "Note title must not be empty.")
	}
	if strings.TrimSpace(body) == "" {
		return nil, qq.failure(http.StatusBadRequest, "Note text must not be empty.")
	}
	return qq.create(ctx, doc, title, body)
}

// Edit preserves original attribution and changes only text and last-editor metadata.
func (qq *DocumentNotes) Edit(
	ctx ctxx.Context, fileID, noteID, title, body string,
) (*enttenant.DocumentNote, error) {
	if strings.TrimSpace(title) == "" {
		return nil, qq.failure(http.StatusBadRequest, "Note title must not be empty.")
	}
	if strings.TrimSpace(body) == "" {
		return nil, qq.failure(http.StatusBadRequest, "Note text must not be empty.")
	}
	note, err := qq.changeable(ctx, fileID, noteID)
	if err != nil {
		return nil, err
	}
	count, err := qq.currentUpdate(ctx, note).SetTitle(title).SetBody(body).
		SetEditedAt(time.Now()).SetEditorID(ctx.TenantCtx().User.ID).Save(ctx)
	if _, err := qq.changed(count, err); err != nil {
		return nil, err
	}
	_, updated, err := qq.Get(ctx, fileID, note.PublicID.String())
	return updated, err
}

// Replace atomically retains the predecessor and creates a separately authored successor.
func (qq *DocumentNotes) Replace(
	ctx ctxx.Context, fileID, noteID, title, body string,
) (*enttenant.DocumentNote, error) {
	if strings.TrimSpace(title) == "" {
		return nil, qq.failure(http.StatusBadRequest, "Note title must not be empty.")
	}
	if strings.TrimSpace(body) == "" {
		return nil, qq.failure(http.StatusBadRequest, "Note text must not be empty.")
	}
	note, err := qq.changeable(ctx, fileID, noteID)
	if err != nil {
		return nil, err
	}
	// Acquire SQLite's writer lock with a conditional no-op before creating a successor.
	// The lock lasts for the caller's transaction, so a stale contender cannot create
	// a competing successor. Attribution is untouched and failures must roll back.
	count, err := qq.currentUpdate(ctx, note).SetBody(note.Body).Save(ctx)
	if _, err := qq.changed(count, err); err != nil {
		return nil, err
	}
	doc, err := qq.document(ctx, fileID)
	if err != nil {
		return nil, err
	}
	successor, err := qq.create(ctx, doc, title, body)
	if err != nil {
		return nil, err
	}
	count, err = qq.currentUpdate(ctx, note).SetReplacedByID(successor.ID).Save(ctx)
	if _, err := qq.changed(count, err); err != nil {
		return nil, err
	}
	return successor, nil
}

// Delete retains the current note as read-only history; it never physically deletes a row.
func (qq *DocumentNotes) Delete(ctx ctxx.Context, fileID, noteID string) (bool, error) {
	note, err := qq.changeable(ctx, fileID, noteID)
	if err != nil {
		return false, err
	}
	count, err := qq.currentUpdate(ctx, note).SetDeletedAt(time.Now()).Save(ctx)
	return qq.changed(count, err)
}

// Transfer reparents all source history within the same Space, retaining target notes.
// It does not edit content and therefore does not require each individual author's permission.
func (qq *DocumentNotes) Transfer(
	ctx ctxx.Context, source, target *enttenant.File,
) (bool, error) {
	if source == nil || target == nil || source.ID == target.ID {
		return false, qq.failure(http.StatusBadRequest, "Source and target must be different files.")
	}
	sourceDoc, err := qq.document(ctx, source.PublicID.String())
	if err != nil {
		return false, err
	}
	targetDoc, err := qq.document(ctx, target.PublicID.String())
	if err != nil {
		return false, err
	}
	if _, err := qq.editable(ctx, sourceDoc); err != nil {
		return false, err
	}
	if _, err := qq.editable(ctx, targetDoc); err != nil {
		return false, err
	}
	if sourceDoc.Notes != "" {
		if _, err := qq.materialize(ctx, sourceDoc); err != nil {
			return false, err
		}
	}
	_, err = ctx.TenantCtx().TTx.DocumentNote.Update().
		Where(documentnote.FileID(sourceDoc.ID)).SetFileID(targetDoc.ID).Save(ctx)
	if err != nil {
		log.Printf("transfer document notes: %v", err)
		return false, err
	}
	return true, nil
}

func (qq *DocumentNotes) document(ctx ctxx.Context, fileID string) (*enttenant.File, error) {
	if _, err := qq.access(ctx); err != nil {
		return nil, err
	}
	doc, err := ctx.TenantCtx().TTx.File.Query().Where(
		file.PublicID(entx.NewCIText(fileID)), file.SpaceID(ctx.SpaceCtx().Space.ID),
		file.IsDirectory(false),
	).Only(schema.SkipSoftDelete(ctx))
	if err != nil {
		log.Printf("read notes document: %v", err)
		return nil, qq.failure(http.StatusNotFound, "Document not found.")
	}
	return doc, nil
}

func (qq *DocumentNotes) access(ctx ctxx.Context) (bool, error) {
	if ctx == nil || !ctx.IsSpaceCtx() || ctx.SpaceCtx().Space == nil ||
		ctx.TenantCtx().User == nil || ctx.TenantCtx().TTx == nil ||
		ctx.MainCtx().Account == nil || ctx.TenantCtx().Tenant == nil {
		return false, qq.failure(http.StatusForbidden, "You cannot access this document's notes.")
	}
	membership, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().Where(
		tenantaccountassignment.AccountID(ctx.MainCtx().Account.ID),
		tenantaccountassignment.TenantID(ctx.TenantCtx().Tenant.ID),
	).Only(ctx)
	if err != nil || membership.ExpiresAt != nil && !membership.ExpiresAt.After(time.Now()) {
		log.Printf("verify note tenant membership: %v", err)
		return false, qq.failure(http.StatusForbidden, "You cannot access this document's notes.")
	}
	// Re-read membership, rather than trusting the role cached when the context was built.
	actor, err := ctx.TenantCtx().TTx.User.Query().Where(
		user.ID(ctx.TenantCtx().User.ID), user.AccountID(ctx.MainCtx().Account.ID),
	).Only(ctx)
	if err != nil {
		log.Printf("verify note actor: %v", err)
		return false, qq.failure(http.StatusForbidden, "You cannot access this document's notes.")
	}
	if _, err := ctx.TenantCtx().TTx.Space.Get(ctx, ctx.SpaceCtx().Space.ID); err != nil {
		log.Printf("verify note Space: %v", err)
		return false, qq.failure(http.StatusForbidden, "You cannot access this document's notes.")
	}
	if actor.Role == tenantrole.Owner {
		return true, nil
	}
	assignment, err := ctx.TenantCtx().TTx.SpaceUserAssignment.Query().Where(
		spaceuserassignment.SpaceID(ctx.SpaceCtx().Space.ID),
		spaceuserassignment.UserID(actor.ID),
	).Only(ctx)
	if err != nil {
		log.Printf("verify note Space membership: %v", err)
		return false, qq.failure(http.StatusForbidden, "You cannot access this document's notes.")
	}
	return assignment.Role == spacerole.Owner, nil
}

func (qq *DocumentNotes) editable(ctx ctxx.Context, doc *enttenant.File) (bool, error) {
	if !doc.DeletedAt.IsZero() || ctx.TenantCtx().IsReadOnlyTx() {
		return false, qq.failure(http.StatusForbidden, "Notes in Trash are read-only.")
	}
	return true, nil
}

func (qq *DocumentNotes) changeable(
	ctx ctxx.Context, fileID, noteID string,
) (*enttenant.DocumentNote, error) {
	doc, note, err := qq.Get(ctx, fileID, noteID)
	if err != nil {
		return nil, err
	}
	if _, err := qq.editable(ctx, doc); err != nil {
		return nil, err
	}
	authorID := int64(0)
	if note != nil {
		authorID = note.AuthorID
		if note.DeletedAt != nil || note.ReplacedByID != 0 {
			return nil, qq.failure(http.StatusConflict, "Historical notes cannot be changed.")
		}
	}
	if !qq.CanChange(ctx, doc, authorID) {
		return nil, qq.failure(http.StatusForbidden, "You cannot change this note.")
	}
	if note == nil {
		return qq.materialize(ctx, doc)
	}
	return note, nil
}

func (qq *DocumentNotes) create(
	ctx ctxx.Context, doc *enttenant.File, title, body string,
) (*enttenant.DocumentNote, error) {
	note, err := ctx.TenantCtx().TTx.DocumentNote.Create().SetSpaceID(doc.SpaceID).
		SetFileID(doc.ID).SetTitle(title).SetBody(body).SetAuthorID(ctx.TenantCtx().User.ID).
		SetAuthoredAt(time.Now()).Save(ctx)
	if err != nil {
		log.Printf("create document note: %v", err)
	}
	return note, err
}

func (qq *DocumentNotes) materialize(
	ctx ctxx.Context, doc *enttenant.File,
) (*enttenant.DocumentNote, error) {
	// Claim the exact legacy value before inserting, so retries cannot duplicate it.
	count, err := ctx.TenantCtx().TTx.File.Update().Where(
		file.ID(doc.ID), file.Notes(doc.Notes),
	).ClearNotes().Save(ctx)
	if _, err := qq.changed(count, err); err != nil {
		return nil, err
	}
	note, err := ctx.TenantCtx().TTx.DocumentNote.Create().SetSpaceID(doc.SpaceID).
		SetFileID(doc.ID).SetBody(doc.Notes).Save(ctx)
	if err != nil {
		log.Printf("materialize legacy document note: %v", err)
	}
	return note, err
}

func (qq *DocumentNotes) currentUpdate(
	ctx ctxx.Context, note *enttenant.DocumentNote,
) *enttenant.DocumentNoteUpdate {
	return ctx.TenantCtx().TTx.DocumentNote.Update().Where(
		documentnote.ID(note.ID), documentnote.FileID(note.FileID),
		documentnote.DeletedAtIsNil(), documentnote.ReplacedByIDIsNil(),
	)
}

func (qq *DocumentNotes) changed(count int, err error) (bool, error) {
	if err != nil {
		log.Printf("change document note: %v", err)
		return false, err
	}
	if count != 1 {
		return false, qq.failure(http.StatusConflict, "Note has already changed. Please reload.")
	}
	return true, nil
}

func (qq *DocumentNotes) failure(status int, message string) error {
	log.Printf("document notes: %s", message)
	return e.NewHTTPErrorf(status, message)
}
