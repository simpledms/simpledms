package managetags

import (
	autil "github.com/marcobeierer/go-core/action/util"

	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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

func (qq *ManageTagsPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T("Tags"),
		},
		Actions: []widget.IWidget{},
	}
}
