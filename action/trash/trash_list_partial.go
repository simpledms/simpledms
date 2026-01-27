package trash

import (
	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *TrashListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

	var listItems []wx.IWidget
	for _, filex := range deletedFiles {
		isSelected := data != nil && filex.PublicID.String() == data.SelectedFileID
		listItems = append(listItems, qq.listItem(ctx, filex, isSelected))
	}

	var content wx.IWidget
	content = &wx.List{
		Children: listItems,
	}

	if len(listItems) == 0 {
		content = &wx.EmptyState{
			Icon:     wx.NewIcon("delete"),
			Headline: wx.T("Trash is empty."),
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.ListID(),
		},
		Children: content,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.Endpoint(),
			HxTrigger: event.HxTrigger(
				event.FileRestored,
			),
			HxTarget: "#" + qq.ListID(),
			HxSwap:   "outerHTML",
		},
	}
}

func (qq *TrashListPartial) listItem(ctx ctxx.Context, filex *enttenant.File, isSelected bool) *wx.ListItem {
	icon := wx.NewIcon("description")
	headline := wx.Tu(filex.Name)

	if filex.IsDirectory {
		icon = wx.NewIcon("folder")
	} else {
		filem := model.NewFile(filex)
		headline = wx.Tu(filem.FilenameInApp(ctx, false))
	}

	var deletedAt *wx.Text
	if filex.DeletedAt.IsZero() {
		if filex.IsDirectory {
			deletedAt = wx.T("Folder deleted")
		} else {
			deletedAt = wx.T("Deleted")
		}
	} else {
		if filex.IsDirectory {
			deletedAt = wx.Tf("Folder deleted on %s", filex.DeletedAt.Format("02 Jan 2006"))
		} else {
			deletedAt = wx.Tf("Deleted on %s", filex.DeletedAt.Format("02 Jan 2006"))
		}
	}

	item := &wx.ListItem{
		RadioGroupName: "trashListRadioGroup",
		RadioValue:     filex.PublicID.String(),
		Leading:        icon.SmallPadding(),
		Headline:       headline,
		SupportingText: deletedAt,
		IsSelected:     isSelected,
	}

	if !filex.IsDirectory {
		item.ContextMenu = qq.actions.TrashContextMenuPartial.Widget(ctx, filex)
		item.HTMXAttrs = wx.HTMXAttrs{
			HxTarget: "#details",
			HxSwap:   "outerHTML",
			HxGet:    route.TrashFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.PublicID.String()),
		}
	}

	return item
}
