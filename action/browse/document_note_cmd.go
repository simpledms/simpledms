package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type DocumentNoteCmd struct {
	notes *filemodel.DocumentNotes
	*actionx.Config
}

func NewDocumentNoteCmd(infra *common.Infra, actions *Actions) *DocumentNoteCmd {
	qq := new(DocumentNoteCmd)
	qq.notes = filemodel.NewDocumentNotes()
	qq.Config = actionx.NewConfig(actions.Route("document-note-cmd"), false)
	return qq
}

func (qq *DocumentNoteCmd) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[DocumentNoteCmdData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	var snackbar *widget.Snackbar
	switch data.Operation {
	case "create":
		_, err = qq.notes.Create(ctx, data.FileID, data.Title, data.Body)
		snackbar = widget.NewSnackbarf("Note added.")
	case "edit":
		_, err = qq.notes.Edit(ctx, data.FileID, data.NoteID, data.Title, data.Body)
		snackbar = widget.NewSnackbarf("Note updated.")
	case "replace":
		_, err = qq.notes.Replace(ctx, data.FileID, data.NoteID, data.Title, data.Body)
		snackbar = widget.NewSnackbarf("Note replaced.")
	case "delete":
		_, err = qq.notes.Delete(ctx, data.FileID, data.NoteID)
		snackbar = widget.NewSnackbarf("Note deleted.")
	default:
		err = e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported note operation.")
	}
	if err != nil {
		log.Println(err)
		return err
	}
	rw.AddRenderables(snackbar)
	// The submitting dialog closes itself after settlement. The global closeDialog
	// event would also close the document's Details sheet on compact screens.
	rw.Header().Set("HX-Trigger", documentNotesChanged)
	return nil
}
