package inbox

// package action

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type FileListItemPartialData struct {
	FileID int64
}

type FileListItemPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileListItemPartial(infra *common.Infra, actions *Actions) *FileListItemPartial {
	return &FileListItemPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-list-item-partial"),
			true,
		),
	}
}

func (qq *FileListItemPartial) Data(fileID int64) *FileListItemPartialData {
	return &FileListItemPartialData{
		FileID: fileID,
	}
}

func (qq *FileListItemPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileListItemPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := ctx.TenantCtx().TTx.File.Query().WithChildren().Where(file.ID(data.FileID)).OnlyX(ctx)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		// TODO is hrefFn correct?
		qq.Widget(ctx, route.Inbox, filex, false), // TODO isSelected via state or data?
	)
}

type HrefFn func(tenantID, spaceID, fileID string) string

func (qq *FileListItemPartial) Widget(
	ctx ctxx.Context,
	hrefFn HrefFn,
	// listState *ListFilesPartialState,
	fileWithChildren *enttenant.File,
	isSelected bool,
) *widget.ListItem {
	/*trailing := &IconButton{
		Icon:     "more_vert",
		Children: NewFileContextMenuWidget(qq.actions).Widget(fileWithChildren),
	}*/

	htmxAttrs := widget.HTMXAttrs{
		HxTarget:  "#details",
		HxSwap:    "outerHTML",
		HxGet:     hrefFn(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, fileWithChildren.PublicID.String()),
		HxHeaders: autil.PreserveStateHeader(),
	}

	return &widget.ListItem{
		RadioGroupName: "fileListRadioGroup",
		Leading:        widget.NewIcon("description").SmallPadding(),
		Headline:       widget.T(fileWithChildren.Name),
		/*SupportingText: wx.Tf(
			"%s, %s",
			qq.infra.FileRepo.GetXX(fileWithChildren).CurrentVersion(ctx).SizeString(),
			fileWithChildren.ModifiedAt.Format("02. January 06"),
		),*/
		HTMXAttrs: htmxAttrs,
		// Trailing:   trailing,
		IsSelected:  isSelected,
		ContextMenu: NewFileContextMenuWidget(qq.actions).Widget(ctx, fileWithChildren),
	}
}
