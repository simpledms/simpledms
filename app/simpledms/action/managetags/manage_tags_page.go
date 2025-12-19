package managetags

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/ui/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type ManageTagsPageState struct {
	TagListState // TODO not embedded and flatten? or could this lead to conflicts?
}

type ManageTagsPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewManageTagsPage(infra *common.Infra, actions *Actions) *ManageTagsPage {
	return &ManageTagsPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *ManageTagsPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[ManageTagsPageState](rw, req)

	/*
		tagIDStr := req.PathValue("tag_id")
		tagID := 0
		if tagIDStr != "" {
			var err error
			tagID, err = strconv.Atoi(tagIDStr)
			if err != nil {
				return e.NewHTTPErrorf(http.StatusBadRequest, "Could not convert id to integer.")
			}
		}
		// TODO is this safe? should be on 64 bit system
		tagID64 := int64(tagID)

	*/

	return qq.Render(rw, req, ctx, qq.infra, "Manage tags", qq.Widget(ctx, state))
}

func (qq *ManageTagsPage) Widget(
	ctx ctxx.Context,
	state *ManageTagsPageState,
) *wx.MainLayout {
	fabs := []*wx.FloatingActionButton{
		{
			Icon:    "add",
			Tooltip: wx.T("Create new tag or group"),
			HTMXAttrs: qq.actions.Tagging.CreateTag.ModalLinkAttrs(
				qq.actions.Tagging.CreateTag.Data(0), ""),
		},
	}

	return &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "manage-tags", fabs),
		Content: &wx.DefaultLayout{
			AppBar:  qq.appBar(ctx),
			Content: qq.actions.TagList.Widget(ctx, qq.actions.TagList.Data(0), &state.TagListState),
		},
	}
}

func (qq *ManageTagsPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "label",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.T("Tags"),
		},
		Actions: []wx.IWidget{},
	}
}
