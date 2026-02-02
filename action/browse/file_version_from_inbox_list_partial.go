package browse

import (
	"net/http"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionFromInboxListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersionFromInboxListPartial(infra *common.Infra, actions *Actions) *FileVersionFromInboxListPartial {
	config := actionx.NewConfig(actions.Route("file-version-from-inbox-list-partial"), true)
	return &FileVersionFromInboxListPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionFromInboxListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionFromInboxDialogData](rw, req, ctx)
	if err != nil {
		return err
	}

	files, err := qq.listFiles(ctx, data)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.listWrapper(ctx, data, files),
	)
}

func (qq *FileVersionFromInboxListPartial) listFiles(ctx ctxx.Context, data *FileVersionFromInboxDialogData) ([]*enttenant.File, error) {
	if data.TargetFileID == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Target file is required.")
	}

	query := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.SpaceID(ctx.SpaceCtx().Space.ID),
			file.IsInInbox(true),
			file.IsDirectory(false),
			file.DeletedAtIsNil(),
		)

	if data.SearchQuery != "" {
		query = query.Where(file.NameContains(data.SearchQuery))
	}

	query = query.Order(file.ByName(sql.OrderAsc()))

	return query.All(ctx)
}

func (qq *FileVersionFromInboxListPartial) findInboxFile(ctx ctxx.Context, sourceFileID string) (*enttenant.File, error) {
	if sourceFileID == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is required.")
	}

	filex := qq.infra.FileRepo.GetX(ctx, sourceFileID)
	if !filex.Data.IsInInbox {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File must be in inbox.")
	}

	return filex.Data, nil
}

func (qq *FileVersionFromInboxListPartial) listWrapper(ctx ctxx.Context, data *FileVersionFromInboxDialogData, files []*enttenant.File) *wx.Container {
	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.actions.FileVersionFromInboxDialog.listID(),
		},
		Child: &wx.List{Children: qq.listItems(ctx, data, files)},
	}
}

func (qq *FileVersionFromInboxListPartial) listItems(ctx ctxx.Context, data *FileVersionFromInboxDialogData, files []*enttenant.File) []wx.IWidget {
	if len(files) == 0 {
		return []wx.IWidget{
			&wx.ListItem{
				Headline: wx.T("No matches found."),
				Type:     wx.ListItemTypeHelper,
			},
		}
	}

	items := make([]wx.IWidget, 0, len(files))
	for _, filex := range files {
		listItem := &wx.ListItem{
			Headline:       wx.T(filex.Name),
			IsSelected:     filex.PublicID.String() == data.SourceFileID,
			RadioGroupName: "SourceFileID",
			RadioValue:     filex.PublicID.String(),
		}
		items = append(items, listItem)
	}

	return items
}
