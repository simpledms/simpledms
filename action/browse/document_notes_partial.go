package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/user"
	"github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

const documentNotesChanged = "documentNotesChanged"

type DocumentNotesPartial struct {
	infra   *common.Infra
	actions *Actions
	notes   *filemodel.DocumentNotes
	*actionx.Config
}

func NewDocumentNotesPartial(infra *common.Infra, actions *Actions) *DocumentNotesPartial {
	return &DocumentNotesPartial{
		infra:   infra,
		actions: actions,
		notes:   filemodel.NewDocumentNotes(),
		Config:  actionx.NewConfig(actions.Route("document-notes-partial"), true),
	}
}

func (qq *DocumentNotesPartial) Data(fileID string) *DocumentNotesPartialData {
	return &DocumentNotesPartialData{
		FileID: fileID,
	}
}

func (qq *DocumentNotesPartial) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[DocumentNotesPartialData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	doc, notes, err := qq.notes.List(ctx, data.FileID, data.ShowHistory)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(rw, ctx, qq.content(ctx, data, doc, notes))
}

func (qq *DocumentNotesPartial) Widget(
	ctx ctxx.Context, data *DocumentNotesPartialData,
) *widget.ScrollableContent {
	doc, notes, err := qq.notes.List(ctx, data.FileID, data.ShowHistory)
	if err != nil {
		log.Println(err)
		return &widget.ScrollableContent{
			Widget: widget.Widget[widget.ScrollableContent]{
				ID: "documentNotes",
			},
			MarginY:  true,
			Children: widget.NewBody(widget.BodyTypeSm, widget.T("Could not load notes.")),
		}
	}
	return qq.content(ctx, data, doc, notes)
}

func (qq *DocumentNotesPartial) content(
	ctx ctxx.Context, data *DocumentNotesPartialData,
	doc *enttenant.File, notes []*enttenant.DocumentNote,
) *widget.ScrollableContent {
	content := &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: "documentNotes",
		},
		MarginY: true,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.Endpoint(),
			// Refreshes retain the rendered state; the history button sends its inverse.
			HxVals:    util.JSON(data),
			HxTarget:  "#documentNotes",
			HxSwap:    "outerHTML",
			HxSync:    "#documentNotes:replace",
			HxTrigger: documentNotesChanged + " from:body",
		},
	}
	history := &widget.IconButton{
		Widget: widget.Widget[widget.IconButton]{
			ID: "documentNotesHistory",
		},
		Icon:       "history",
		Label:      widget.T("Show deleted and replaced notes"),
		Tooltip:    widget.T("Show deleted and replaced notes"),
		IsToggle:   true,
		IsSelected: data.ShowHistory,
		HTMXAttrs: widget.HTMXAttrs{
			Role:   "button",
			HxPost: qq.Endpoint(),
			HxVals: util.JSON(&DocumentNotesPartialData{
				FileID:      data.FileID,
				ShowHistory: !data.ShowHistory,
			}),
			HxTarget: "#documentNotes",
			HxSwap:   "outerHTML",
			HxSync:   "#documentNotes:replace",
		},
	}
	toolbar := []widget.IWidget{}
	if doc.DeletedAt.IsZero() {
		toolbar = append(toolbar, &widget.IconButton{
			Icon:    "add",
			Label:   widget.T("Add note"),
			Tooltip: widget.T("Add note"),
			HTMXAttrs: widget.HTMXAttrs{
				Role:          "button",
				HxPost:        qq.actions.DocumentNoteDialog.Endpoint(),
				HxVals:        util.JSON(qq.actions.DocumentNoteDialog.Data(data.FileID, "", "create")),
				LoadInPopover: true,
			},
		})
	}
	toolbar = append(toolbar, history)
	content.Toolbar = widget.NewToolbar(widget.T("Notes").String(ctx), toolbar...)
	entries := []widget.IWidget{}
	canChangeOthers := qq.notes.CanChange(ctx, doc, 0)
	for _, note := range notes {
		entries = append(entries, qq.entry(ctx, doc, note, canChangeOthers))
	}
	if doc.Notes != "" {
		entries = append(entries, qq.entry(ctx, doc, nil, canChangeOthers))
	}
	if len(entries) == 0 {
		empty := &widget.EmptyState{
			Headline: widget.T("No notes available."),
		}
		if doc.DeletedAt.IsZero() {
			empty.Actions = []widget.IWidget{&widget.Button{
				Label:     widget.T("Add note"),
				Icon:      widget.NewIcon("add"),
				StyleType: widget.ButtonStyleTypeElevated,
				HTMXAttrs: widget.HTMXAttrs{
					Role:          "button",
					HxPost:        qq.actions.DocumentNoteDialog.Endpoint(),
					HxVals:        util.JSON(qq.actions.DocumentNoteDialog.Data(data.FileID, "", "create")),
					LoadInPopover: true,
				},
			}}
		}
		content.Children = empty
	} else {
		content.Children = &widget.List{
			Children: entries,
		}
	}
	return content
}

func (qq *DocumentNotesPartial) noteContent(
	ctx ctxx.Context, doc *enttenant.File, note *enttenant.DocumentNote,
) *partial.DocumentNoteContent {
	entry := &partial.DocumentNoteContent{
		Title:    widget.T("Note"),
		Body:     doc.Notes,
		Metadata: []*widget.Text{widget.T("Author: Unknown"), widget.T("Created: Unknown")},
	}
	if note != nil {
		entry.Body = note.Body
		if note.Title != "" {
			entry.Title = widget.Tu(note.Title)
		}
		if note.Edges.Author != nil {
			entry.Metadata[0] = widget.Tf("Author: %s", user.NewUser(note.Edges.Author).Name())
		}
		if note.AuthoredAt != nil {
			entry.Metadata[1] = widget.Tf("Created: %s",
				timex.NewDateTime(*note.AuthoredAt).String(ctx.MainCtx().LanguageBCP47))
		}
		if note.EditedAt != nil {
			editor := widget.T("Unknown").String(ctx)
			if note.Edges.Editor != nil {
				editor = user.NewUser(note.Edges.Editor).Name()
			}
			entry.Tooltip = widget.Tf("Edited by %s: %s", editor,
				timex.NewDateTime(*note.EditedAt).String(ctx.MainCtx().LanguageBCP47))
			entry.Metadata = append(entry.Metadata, entry.Tooltip)
		}
		if note.DeletedAt != nil {
			entry.Status = widget.T("Deleted")
			entry.Metadata = append(entry.Metadata, widget.T("Deleted"))
			entry.Metadata = append(entry.Metadata, widget.Tu(
				timex.NewDateTime(*note.DeletedAt).String(ctx.MainCtx().LanguageBCP47)))
		}
		if note.ReplacedByID != 0 {
			entry.Status = widget.T("Replaced")
			replacementTitle := widget.T("Unknown").String(ctx)
			if note.Edges.Replacement != nil {
				replacementTitle = note.Edges.Replacement.Title
				if replacementTitle == "" {
					replacementTitle = widget.T("Note").String(ctx)
				}
				entry.ReplacementHref = "#documentNote-" + note.Edges.Replacement.PublicID.String()
				entry.ReplacementLabel = widget.T("View replacement note")
				entry.ReplacementAttrs = widget.HTMXAttrs{
					HxPost: qq.actions.DocumentNoteDialog.Endpoint(),
					HxVals: util.JSON(qq.actions.DocumentNoteDialog.Data(
						doc.PublicID.String(), note.Edges.Replacement.PublicID.String(), "view")),
					HxTarget:    "#documentNoteDetailsContent",
					HxSelect:    "#documentNoteDetailsContent",
					HxSelectOOB: "#documentNoteDetailsDialog-headline",
					HxSwap:      "outerHTML",
				}
			}
			entry.ReplacementReference = widget.Tf("Replaced by: %s", replacementTitle)
		}
	}
	return entry
}

func (qq *DocumentNotesPartial) entry(
	ctx ctxx.Context, doc *enttenant.File, note *enttenant.DocumentNote,
	canChangeOthers bool,
) *widget.ListItem {
	entry := qq.noteContent(ctx, doc, note)
	id := "legacy"
	authorID := int64(0)
	isCurrent := true
	if note != nil {
		id = note.PublicID.String()
		authorID = note.AuthorID
		isCurrent = note.DeletedAt == nil && note.ReplacedByID == 0
	}
	entry.IsSummary = true
	entry.Tooltip = widget.Tuf("%s\n%s",
		entry.Metadata[1].String(ctx), entry.Metadata[0].String(ctx))
	item := &widget.ListItem{
		Widget: widget.Widget[widget.ListItem]{
			ID: "documentNote-" + id,
		},
		Content: entry,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:        qq.actions.DocumentNoteDialog.Endpoint(),
			HxVals:        util.JSON(qq.actions.DocumentNoteDialog.Data(doc.PublicID.String(), id, "view")),
			LoadInPopover: true,
		},
	}
	// The list has already checked document access; mutation commands recheck it.
	canChange := canChangeOthers || authorID != 0 && authorID == ctx.TenantCtx().User.ID
	if isCurrent && doc.DeletedAt.IsZero() && canChange {
		menu := &widget.Menu{
			Widget: widget.Widget[widget.Menu]{
				ID: "documentNoteMenu-" + id,
			},
			Position: widget.PositionLeft,
			Items: []*widget.MenuItem{
				qq.menuItem(doc.PublicID.String(), id, "edit", widget.T("Edit")),
				qq.menuItem(doc.PublicID.String(), id, "replace", widget.T("Replace")),
				qq.menuItem(doc.PublicID.String(), id, "delete", widget.T("Delete")),
			},
		}
		item.ContextMenu = menu
	}
	return item
}

func (qq *DocumentNotesPartial) menuItem(
	fileID, noteID, operation string, label *widget.Text,
) *widget.MenuItem {
	return &widget.MenuItem{
		Label: label,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:        qq.actions.DocumentNoteDialog.Endpoint(),
			HxVals:        util.JSON(qq.actions.DocumentNoteDialog.Data(fileID, noteID, operation)),
			LoadInPopover: true,
		},
	}
}
