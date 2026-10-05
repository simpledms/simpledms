package inbox

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

// FileContextMenuPartial renders the items of a file row's context menu when it opens, so file
// lists neither render nor query a menu per row.
type FileContextMenuPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileContextMenuPartial(infra *common.Infra, actions *Actions) *FileContextMenuPartial {
	return &FileContextMenuPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("file-context-menu-partial"), true),
	}
}

func (qq *FileContextMenuPartial) Data(fileID string) *FileContextMenuPartialData {
	return &FileContextMenuPartialData{
		FileID: fileID,
	}
}

func (qq *FileContextMenuPartial) LazyMenu(filex *enttenant.File) *widget.Menu {
	return &widget.Menu{
		LazyItems: &widget.HTMXAttrs{
			HxPost: qq.Endpoint(),
			HxVals: util.JSON(qq.Data(filex.PublicID.String())),
		},
	}
}

func (qq *FileContextMenuPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[FileContextMenuPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex, err := ctx.SpaceCtx().Space.QueryFiles().
		Where(file.PublicID(entx.NewCIText(data.FileID))).
		Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "File not found.")
		}
		log.Println(err)
		return err
	}

	menu := NewFileContextMenuWidget(qq.actions).Widget(ctx, filex)
	return qq.infra.Renderer().Render(rw, ctx, autil.MenuItemRenderables(menu)...)
}
