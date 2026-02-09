package openfile

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
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
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "upload", fabs),
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

	spacesByTenant := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()

	// TODO ordner?
	for tenantx, spaces := range spacesByTenant {
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
					},
				})
			}
		}
	}

	var fabs []*wx.FloatingActionButton

	mainLayout := &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "upload", fabs),
		Content: &wx.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List: &wx.List{
				Children: spaceItems,
			},
		},
	}

	return mainLayout
}

func (qq *SelectSpacePage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "upload", // TODO hub or upload?
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &wx.AppBarTitle{
			Text: wx.T("Select space"),
		},
		Actions: []wx.IWidget{},
	}
}
