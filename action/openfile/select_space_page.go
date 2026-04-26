package openfile

import (
	"log"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/simpledms/simpledms/ui/uix/partial"

	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type SelectSpacePageState struct {
	// Cmd string
}

type SelectSpacePage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
	// cache     *cache.Cache
	tenantDBs *tenantdbs.TenantDBs
}

func NewSelectSpacePage(
	infra *common.Infra,
	actions *Actions,
	// cachex *cache.Cache,
	tenantDBs *tenantdbs.TenantDBs,
) *SelectSpacePage {
	return &SelectSpacePage{
		infra:   infra,
		actions: actions,
		// cache:     cachex,
		tenantDBs: tenantDBs,
	}
}

func (qq *SelectSpacePage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	state := autil.StateX[SelectSpacePageState](rw, req)

	token := req.PathValue("upload_token")
	if token == "" {
		return qq.Render(rw, req, ctx, qq.infra, "Upload", qq.WaitWidget(ctx, token, state))
		// return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid token.")
	}

	widget, err := qq.Widget(ctx, token, state)
	if err != nil {
		log.Println(err)
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "Upload", widget)
}

func (qq *SelectSpacePage) WaitWidget(ctx ctxx.Context, uploadToken string, state *SelectSpacePageState) renderable.Renderable {
	fabs := []*widget.FloatingActionButton{}

	// TODO make clear if nothing happens, if there are no files...
	return &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "upload", fabs),
		Content: &widget.ListDetailLayout{ // TODO implement a FullPageLayout instead
			AppBar: qq.appBar(ctx),
			List: []widget.IWidget{
				&widget.EmptyState{
					Icon:        widget.NewIcon("upload"),
					Headline:    widget.T("Uploading files, please wait a moment."),
					Description: widget.T("The page will be refreshed automatically once the upload is finished."),
				},
			},
		},
	}
}

func (qq *SelectSpacePage) Widget(
	ctx ctxx.Context,
	uploadToken string,
	state *SelectSpacePageState,
) (renderable.Renderable, error) {
	// TODO show files or just stats (number of files and total size)?
	// TODO redirect to inbox of space (preselect first file)

	// TODO autoselect space if just one?

	var spaceItems []*widget.ListItem

	spacesByTenant, err := ctx.AppCtx().ReadOnlyAccountSpacesByTenant()
	if err != nil {
		return nil, err
	}

	// TODO ordner?
	for tenantx, spaces := range spacesByTenant {
		if len(spaces) == 0 {
			spaceItems = append(spaceItems, &widget.ListItem{
				Type:           widget.ListItemTypeHelper,
				Headline:       widget.T("No spaces yet."),
				SupportingText: widget.T("Please try again once you created a space or were invited to join one."),
			})
		} else {
			for _, spacex := range spaces {
				spaceItems = append(spaceItems, &widget.ListItem{
					Headline:       widget.Tu(spacex.Name),
					SupportingText: widget.Tu(tenantx.Name),
					HTMXAttrs: widget.HTMXAttrs{
						// redirecting to inbox instead of using a custom action like SelectSpace because
						// this way we get the security check for space and tenant for free and don't have
						// to be very careful in the custom action
						// FIXME make type safe
						// HxGet: route.InboxRoot(tenantx.PublicID.String(), spacex.PublicID.String()) + "?upload_token=" + uploadToken,
						HxGet: route.InboxRootWithState(struct {
							UploadToken string `url:"upload_token"`
						}{
							UploadToken: uploadToken,
						})(tenantx.PublicID.String(), spacex.PublicID.String()),
					},
				})
			}
		}
	}

	var fabs []*widget.FloatingActionButton

	mainLayout := &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "upload", fabs),
		Content: &widget.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List: &widget.List{
				Children: spaceItems,
			},
		},
	}

	return mainLayout, nil
}

func (qq *SelectSpacePage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "upload", // TODO hub or upload?
		},
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T("Select space"),
		},
		Actions: []widget.IWidget{},
	}
}
