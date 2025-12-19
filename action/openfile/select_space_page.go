package openfile

import (
	"log"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/renderable"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/partial"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *SelectSpacePage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[SelectSpacePageState](rw, req)

	token := req.PathValue("upload_token")
	if token == "" {
		return qq.Render(rw, req, ctx, qq.infra, "Upload", qq.WaitWidget(ctx, token, state))
		// return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid token.")
	}

	return qq.Render(rw, req, ctx, qq.infra, "Upload", qq.Widget(ctx, token, state))
}

func (qq *SelectSpacePage) WaitWidget(ctx ctxx.Context, uploadToken string, state *SelectSpacePageState) renderable.Renderable {
	fabs := []*wx.FloatingActionButton{}

	// TODO make clear if nothing happens, if there are no files...
	return &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "upload", fabs),
		Content: &wx.ListDetailLayout{ // TODO implement a FullPageLayout instead
			AppBar: qq.appBar(ctx),
			List: []wx.IWidget{
				&wx.EmptyState{
					Icon:        wx.NewIcon("upload"),
					Headline:    wx.T("Uploading files, please wait a moment."),
					Description: wx.T("The page will be refreshed automatically once the upload is finished."),
				},
			},
		},
	}
}

func (qq *SelectSpacePage) Widget(ctx ctxx.Context, uploadToken string, state *SelectSpacePageState) renderable.Renderable {
	// TODO show files or just stats (number of files and total size)?
	// TODO redirect to inbox of space (preselect first file)

	// TODO autoselect space if just one?

	var spaceItems []*wx.ListItem

	// TODO add all spaces
	//		spaces list (tenant name)

	tenants := ctx.MainCtx().Account.QueryTenants().AllX(ctx)

	for _, tenantx := range tenants {
		tenantDB, ok := qq.tenantDBs.Load(tenantx.ID)
		if !ok {
			log.Println("tenant db not found, tenant id was", tenantx.ID)
			continue
		}
		spaces, err := tenantDB.ReadOnlyConn.Space.Query().All(ctx)
		if err != nil && !enttenant.IsNotFound(err) {
			log.Println("failed to query spaces for tenant", tenantx.ID, err)
			continue
		}
		if len(spaces) == 0 {
			spaceItems = append(spaceItems, &wx.ListItem{
				Type:           wx.ListItemTypeHelper,
				Headline:       wx.T("No spaces yet."),
				SupportingText: wx.T("Please try again once you created a space or were invited to join one."),
			})
		} else {
			for _, spacex := range spaces {
				spaceItems = append(spaceItems, &wx.ListItem{
					Headline:       wx.Tu(spacex.Name),
					SupportingText: wx.Tu(tenantx.Name),
					HTMXAttrs: wx.HTMXAttrs{
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
						// HxPost: qq.actions.SelectSpace.Endpoint(),
						// HxVals: util.JSON(qq.actions.SelectSpace.Data()),
					},
				})
			}
		}

	}

	spaceList := &wx.List{
		Children: spaceItems,
	}

	var children []wx.IWidget

	children = append(children,
		spaceList,
	)

	fabs := []*wx.FloatingActionButton{}

	mainLayout := &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "upload", fabs),
		Content: &wx.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List:   children,
		},
	}

	return mainLayout
}

func (qq *SelectSpacePage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "upload", // TODO hub or upload?
		},
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.T("Select space"),
		},
		Actions: []wx.IWidget{},
	}
}
