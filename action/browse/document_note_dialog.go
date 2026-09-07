package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type DocumentNoteDialog struct {
	infra   *common.Infra
	actions *Actions
	notes   *filemodel.DocumentNotes
	*actionx.Config
}

func NewDocumentNoteDialog(infra *common.Infra, actions *Actions) *DocumentNoteDialog {
	qq := new(DocumentNoteDialog)
	qq.infra = infra
	qq.actions = actions
	qq.notes = filemodel.NewDocumentNotes()
	qq.Config = actionx.NewConfig(actions.Route("document-note-dialog"), true)
	return qq
}

func (qq *DocumentNoteDialog) Data(fileID, noteID, operation string) *DocumentNoteDialogData {
	data := new(DocumentNoteDialogData)
	data.FileID = fileID
	data.NoteID = noteID
	data.Operation = operation
	return data
}

func (qq *DocumentNoteDialog) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[DocumentNoteDialogData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	body := ""
	title := ""
	if data.Operation == "view" {
		doc, note, err := qq.notes.Get(ctx, data.FileID, data.NoteID)
		if err != nil {
			return err
		}
		content := qq.actions.DocumentNotesPartial.noteContent(ctx, doc, note)
		content.ID = "documentNoteDetailsContent"
		return qq.infra.Renderer().Render(rw, ctx, &widget.Dialog{
			Widget: widget.Widget[widget.Dialog]{
				ID: "documentNoteDetailsDialog",
			},
			Headline:     content.Title,
			IsOpenOnLoad: true,
			Child:        content,
		})
	}
	if data.Operation == "create" {
		doc, _, err := qq.notes.List(ctx, data.FileID, false)
		if err != nil {
			return err
		}
		if !doc.DeletedAt.IsZero() {
			log.Println("cannot add a note in Trash")
			return e.NewHTTPErrorf(http.StatusForbidden, "Notes in Trash are read-only.")
		}
	} else {
		doc, note, err := qq.notes.Get(ctx, data.FileID, data.NoteID)
		if err != nil {
			return err
		}
		authorID := int64(0)
		body = doc.Notes
		if note != nil {
			authorID = note.AuthorID
			body = note.Body
			title = note.Title
			if note.DeletedAt != nil || note.ReplacedByID != 0 {
				log.Println("cannot change a historical note")
				return e.NewHTTPErrorf(http.StatusConflict, "Historical notes cannot be changed.")
			}
		}
		if !qq.notes.CanChange(ctx, doc, authorID) {
			log.Println("note change permission denied")
			return e.NewHTTPErrorf(http.StatusForbidden, "You cannot change this note.")
		}
	}
	var heading, submit *widget.Text
	switch data.Operation {
	case "create":
		heading, submit = widget.T("Add note"), widget.T("Save")
	case "edit":
		heading, submit = widget.T("Edit note"), widget.T("Save")
	case "replace":
		heading, submit = widget.T("Replace note"), widget.T("Replace")
	case "delete":
		heading, submit = widget.T("Delete note"), widget.T("Delete")
	default:
		log.Println("unsupported document note operation", data.Operation)
		return e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported note operation.")
	}
	form := new(widget.Form)
	form.ID = "documentNoteForm"
	form.HxPost = qq.actions.DocumentNoteCmd.Endpoint()
	form.HxVals = util.JSON(data)
	// Errors are rendered by wrapTx; no swap preserves the open dialog and typed text.
	form.HxSwap = "none"
	if data.Operation == "delete" {
		form.Children = []widget.IWidget{widget.NewBody(widget.BodyTypeMd,
			widget.T("Delete this note? It will remain available in the note history."))}
	} else {
		titleField := &widget.TextField{
			Widget: widget.Widget[widget.TextField]{
				ID: "documentNoteTitle",
			},
			Name:         "Title",
			Label:        widget.T("Title"),
			DefaultValue: title,
			HasAutofocus: true,
			IsRequired:   true,
		}
		text := new(widget.TextArea)
		text.ID = "documentNoteBody"
		text.Name = "Body"
		text.Value = body
		text.Label = widget.T("Note")
		form.Children = []widget.IWidget{
			titleField,
			text,
		}
	}
	dialog := new(widget.Dialog)
	dialog.ID = "documentNoteDialog"
	dialog.Headline = heading
	dialog.SubmitLabel = submit
	dialog.FormID = form.ID
	dialog.IsOpenOnLoad = true
	dialog.Child = form
	return qq.infra.Renderer().Render(rw, ctx, dialog)
}
