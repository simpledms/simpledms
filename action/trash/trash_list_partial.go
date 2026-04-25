package trash

import (
	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type TrashListPartialData struct {
	SelectedFileID string
}

type TrashListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewTrashListPartial(infra *common.Infra, actions *Actions) *TrashListPartial {
	return &TrashListPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("trash-list-partial"),
			true,
		),
	}
}

func (qq *TrashListPartial) Data(selectedFileID string) *TrashListPartialData {
	return &TrashListPartialData{SelectedFileID: selectedFileID}
}

func (qq *TrashListPartial) ListID() string {
	return "trashList"
}

func (qq *TrashListPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[TrashListPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *TrashListPartial) Widget(ctx ctxx.Context, data *TrashListPartialData) renderable.Renderable {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)

	deletedFiles := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.SpaceID(ctx.SpaceCtx().Space.ID),
			file.DeletedAtNotNil(),
			file.IsDirectory(false),
		).
		Order(file.ByDeletedAt(sql.OrderDesc())).
		AllX(ctxWithDeleted)

	var listItems []widget.IWidget
	for _, filex := range deletedFiles {
		isSelected := data != nil && filex.PublicID.String() == data.SelectedFileID
		listItems = append(listItems, qq.listItem(ctx, filex, isSelected))
	}

	var content widget.IWidget
	content = &widget.List{
		Children: listItems,
	}

	if len(listItems) == 0 {
		content = &widget.EmptyState{
			Icon:     widget.NewIcon("delete"),
			Headline: widget.T("Trash is empty."),
		}
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.ListID(),
		},
		Children: content,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.Endpoint(),
			HxTrigger: events.HxTrigger(
				event.FileRestored,
			),
			HxTarget: "#" + qq.ListID(),
			HxSwap:   "outerHTML",
		},
	}
}

func (qq *TrashListPartial) listItem(ctx ctxx.Context, filex *enttenant.File, isSelected bool) *widget.ListItem {
	icon := widget.NewIcon("description")
	headline := widget.Tu(filex.Name)

	if filex.IsDirectory {
		icon = widget.NewIcon("folder")
	} else {
		filem := filemodel.NewFile(filex)
		headline = widget.Tu(filem.FilenameInApp(ctx, false))
	}

	var deletedAt *widget.Text
	if filex.DeletedAt.IsZero() {
		if filex.IsDirectory {
			deletedAt = widget.T("Folder deleted")
		} else {
			deletedAt = widget.T("Deleted")
		}
	} else {
		if filex.IsDirectory {
			deletedAt = widget.Tf("Folder deleted on %s", filex.DeletedAt.Format("02 Jan 2006"))
		} else {
			deletedAt = widget.Tf("Deleted on %s", filex.DeletedAt.Format("02 Jan 2006"))
		}
	}

	item := &widget.ListItem{
		RadioGroupName: "trashListRadioGroup",
		RadioValue:     filex.PublicID.String(),
		Leading:        icon.SmallPadding(),
		Headline:       headline,
		SupportingText: deletedAt,
		IsSelected:     isSelected,
	}

	if !filex.IsDirectory {
		item.ContextMenu = qq.actions.TrashContextMenuWidget.Widget(ctx, filex)
		item.HTMXAttrs = widget.HTMXAttrs{
			HxTarget:  "#details",
			HxSwap:    "outerHTML",
			HxGet:     route.TrashFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.PublicID.String()),
			HxHeaders: autil.PreserveStateHeader(),
		}
	}

	return item
}
