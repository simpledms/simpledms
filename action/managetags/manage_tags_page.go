package managetags

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type ManageTagsPageState struct {
	TagListPartialState // TODO not embedded and flatten? or could this lead to conflicts?
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
) *widget.MainLayout {
	fabs := []*widget.FloatingActionButton{
		{
			Icon:    "add",
			Tooltip: widget.T("Create new tag or group"),
			HTMXAttrs: qq.actions.Tagging.CreateTagCmd.ModalLinkAttrs(
				qq.actions.Tagging.CreateTagCmd.Data(0), ""),
		},
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "tags", fabs),
		Content: &widget.DefaultLayout{
			AppBar:  qq.appBar(ctx),
			Content: qq.actions.TagListPartial.Widget(ctx, qq.actions.TagListPartial.Data(0), &state.TagListPartialState),
		},
	}
}

func (qq *ManageTagsPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "label",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("Tags"),
		},
		Actions: []widget.IWidget{},
	}
}
