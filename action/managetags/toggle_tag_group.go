package managetags

import (
	"net/http"
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ToggleTagGroupData struct {
	TagGroupID int64
}

type ToggleTagGroup struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleTagGroup(infra *common.Infra, actions *Actions) *ToggleTagGroup {
	config := actionx.NewConfig("/toggle-tag-group", true)
	return &ToggleTagGroup{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ToggleTagGroup) Data(tagGroupID int64) *ToggleTagGroupData {
	return &ToggleTagGroupData{
		TagGroupID: tagGroupID,
	}
}

func (qq *ToggleTagGroup) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ToggleTagGroupData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ManageTagsPageState](rw, req)

	if data.TagGroupID == 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Tag group ID is required.")
	}

	tagx := ctx.SpaceCtx().Space.QueryTags().
		WithChildren(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		}).
		Where(tag.ID(data.TagGroupID)).
		OnlyX(ctx)
	if tagx.Type != tagtype.Group {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Tag is not a group.")
	}

	if slices.Contains(state.ExpandedGroups, data.TagGroupID) {
		state.ExpandedGroups = slices.DeleteFunc(state.ExpandedGroups, func(tagID int64) bool {
			return tagID == data.TagGroupID
		})
	} else {
		state.ExpandedGroups = append(state.ExpandedGroups, data.TagGroupID)
	}

	rw.Header().Set("HX-Replace-Url", route.ManageTagsWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))

	return qq.infra.Renderer().Render(rw, ctx,
		qq.actions.TagList.listItem(ctx, &state.TagListState, tagx),
	)
}
