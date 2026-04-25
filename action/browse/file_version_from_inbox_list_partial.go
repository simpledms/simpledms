package browse

import (
	"net/http"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
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

func (qq *FileVersionFromInboxListPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

func (qq *FileVersionFromInboxListPartial) listWrapper(ctx ctxx.Context, data *FileVersionFromInboxDialogData, files []*enttenant.File) *widget.Container {
	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.actions.FileVersionFromInboxDialog.listID(),
		},
		Child: &widget.List{Children: qq.listItems(ctx, data, files)},
	}
}

func (qq *FileVersionFromInboxListPartial) listItems(ctx ctxx.Context, data *FileVersionFromInboxDialogData, files []*enttenant.File) []widget.IWidget {
	if len(files) == 0 {
		return []widget.IWidget{
			&widget.ListItem{
				Headline: widget.T("No matches found."),
				Type:     widget.ListItemTypeHelper,
			},
		}
	}

	items := make([]widget.IWidget, 0, len(files))
	for _, filex := range files {
		listItem := &widget.ListItem{
			Headline:       widget.T(filex.Name),
			IsSelected:     filex.PublicID.String() == data.SourceFileID,
			RadioGroupName: "SourceFileID",
			RadioValue:     filex.PublicID.String(),
		}
		items = append(items, listItem)
	}

	return items
}
