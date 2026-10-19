package mcp

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/e"
)

// noteTools exposes authored document notes.
type noteTools struct{}

func newNoteTools() *noteTools {
	return &noteTools{}
}

func (qq *noteTools) listDocumentNotes(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input DocumentNoteListInput,
) (DocumentNoteListData, error) {
	result := DocumentNoteListData{FileID: input.FileID, Notes: []DocumentNoteData{}}
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return result, err
	}
	limit, err := pageLimit(input.Offset, input.Limit)
	if err != nil {
		return result, err
	}
	doc, query, err := filemodel.NewDocumentNotes().Query(ctx, input.FileID, input.ShowHistory)
	if err != nil {
		return result, err
	}
	notes, err := query.Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	notes, result.HasMore, result.NextOffset = trimPage(notes, input.Offset, limit)
	for _, note := range notes {
		data, err := documentNoteProjection(ctx, cred, doc, note, 0, 1000)
		if err != nil {
			return result, err
		}
		result.Notes = append(result.Notes, data)
	}
	if input.Offset == 0 && doc.Notes != "" {
		legacy, err := documentNoteProjection(ctx, cred, doc, nil, 0, 1000)
		if err != nil {
			return result, err
		}
		result.LegacyNote = &legacy
	}
	return result, nil
}

func (qq *noteTools) getDocumentNote(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input GetDocumentNoteInput,
) (DocumentNoteData, error) {
	if err := nilableDocumentNoteReferenceError(input.DocumentNoteInput); err != nil {
		return DocumentNoteData{}, err
	}
	length := 12000
	if input.Length != nil {
		length = *input.Length
	}
	if input.Offset < 0 || input.Offset > 1000000 || length < 1 || length > 50000 {
		return DocumentNoteData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid note text range.")
	}
	doc, note, err := filemodel.NewDocumentNotes().Get(ctx, input.FileID, input.NoteID)
	if err != nil {
		return DocumentNoteData{}, err
	}
	return documentNoteProjection(ctx, cred, doc, note, input.Offset, length)
}

func documentNoteProjection(
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	doc *enttenant.File,
	note *enttenant.DocumentNote,
	offset, length int,
) (DocumentNoteData, error) {
	data := DocumentNoteData{
		FileID:     doc.PublicID.String(),
		NoteID:     "legacy",
		IsLegacy:   note == nil,
		BodyOffset: offset,
	}
	body, authorID, isCurrent := doc.Notes, int64(0), true
	if note != nil {
		data.NoteID = note.PublicID.String()
		data.Title = note.Title
		data.Author = nilableNoteActorProjection(note.Edges.Author)
		data.AuthoredAt = note.AuthoredAt
		data.Editor = nilableNoteActorProjection(note.Edges.Editor)
		data.EditedAt = note.EditedAt
		data.DeletedAt = note.DeletedAt
		if replacement := note.Edges.Replacement; replacement != nil {
			data.ReplacedByNoteID = replacement.PublicID.String()
		}
		body, authorID = note.Body, note.AuthorID
		isCurrent = note.DeletedAt == nil && note.ReplacedByID == 0
	}
	var err error
	data.Body, data.HasMoreBody, err = filemodel.NewFileReader().TextWindow(body, offset, length)
	if err != nil {
		return DocumentNoteData{}, err
	}
	if data.HasMoreBody {
		next := offset + length
		data.NextBodyOffset = &next
	}
	data.CanChange = !cred.IsReadOnly && isCurrent &&
		filemodel.NewDocumentNotes().CanChange(ctx, doc, authorID)
	return data, nil
}

func nilableNoteActorProjection(actor *enttenant.User) *NoteActorData {
	if actor == nil {
		return nil
	}
	return &NoteActorData{
		UserID: actor.PublicID.String(),
		Name:   strings.TrimSpace(actor.FirstName + " " + actor.LastName),
	}
}

func nilableDocumentNoteReferenceError(input DocumentNoteInput) error {
	if input.FileID == "" || len(input.FileID) > 100 || input.NoteID == "" || len(input.NoteID) > 100 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Document and note IDs are required.")
	}
	return nil
}

func nilableDocumentNoteTextError(title, body string) error {
	if utf8.RuneCountInString(title) > 300 || utf8.RuneCountInString(body) > 50000 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Note title or body is too long.")
	}
	return nil
}

func (qq *noteTools) createDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input CreateDocumentNoteInput,
) (DocumentNoteData, error) {
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return DocumentNoteData{}, err
	}
	if err := nilableDocumentNoteTextError(input.Title, input.Body); err != nil {
		return DocumentNoteData{}, err
	}
	note, err := filemodel.NewDocumentNotes().Create(ctx, input.FileID, input.Title, input.Body)
	if err != nil {
		return DocumentNoteData{}, err
	}
	length := 50000
	return qq.getDocumentNote(requestCtx, ctx, cred, GetDocumentNoteInput{
		DocumentNoteInput: DocumentNoteInput{
			FileID: input.FileID,
			NoteID: note.PublicID.String(),
		},
		Length: &length,
	})
}

func (qq *noteTools) editDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentNoteInput,
) (DocumentNoteData, error) {
	return qq.changeDocumentNote(requestCtx, ctx, cred, input, false)
}

func (qq *noteTools) replaceDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentNoteInput,
) (DocumentNoteData, error) {
	return qq.changeDocumentNote(requestCtx, ctx, cred, input, true)
}

func (qq *noteTools) changeDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentNoteInput,
	isReplacement bool,
) (DocumentNoteData, error) {
	if err := nilableDocumentNoteReferenceError(input.DocumentNoteInput); err != nil {
		return DocumentNoteData{}, err
	}
	if err := nilableDocumentNoteTextError(input.Title, input.Body); err != nil {
		return DocumentNoteData{}, err
	}
	var note *enttenant.DocumentNote
	var err error
	service := filemodel.NewDocumentNotes()
	if isReplacement {
		note, err = service.Replace(ctx, input.FileID, input.NoteID, input.Title, input.Body)
	} else {
		note, err = service.Edit(ctx, input.FileID, input.NoteID, input.Title, input.Body)
	}
	if err != nil {
		return DocumentNoteData{}, err
	}
	length := 50000
	return qq.getDocumentNote(requestCtx, ctx, cred, GetDocumentNoteInput{
		DocumentNoteInput: DocumentNoteInput{
			FileID: input.FileID,
			NoteID: note.PublicID.String(),
		},
		Length: &length,
	})
}

func (qq *noteTools) deleteDocumentNote(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input DocumentNoteInput,
) (DocumentNoteDeletionData, error) {
	if err := nilableDocumentNoteReferenceError(input); err != nil {
		return DocumentNoteDeletionData{}, err
	}
	deleted, err := filemodel.NewDocumentNotes().Delete(ctx, input.FileID, input.NoteID)
	return DocumentNoteDeletionData{
		FileID:  input.FileID,
		NoteID:  input.NoteID,
		Deleted: deleted,
	}, err
}
