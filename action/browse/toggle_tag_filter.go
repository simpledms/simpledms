package browse

import (
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ToggleTagFilterData struct {
	CurrentDirID string
	TagID        int64
}

/*
type ToggleTagFilterState struct {
	CheckedTagIDs []int `url:"tag_ids,omitempty"` // shared with ListFilterTagsState
}
*/

type ToggleTagFilter struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleTagFilter(infra *common.Infra, actions *Actions) *ToggleTagFilter {
	config := actionx.NewConfig(
		actions.Route("toggle-tag-filter"),
		true,
	)
	return &ToggleTagFilter{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ToggleTagFilter) Data(currentDirID string, tagID int64) *ToggleTagFilterData {
	return &ToggleTagFilterData{
		CurrentDirID: currentDirID,
		TagID:        tagID,
	}
}

func (qq *ToggleTagFilter) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ToggleTagFilterData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirState](rw, req)

	if slices.Contains(state.CheckedTagIDs, int(data.TagID)) {
		state.CheckedTagIDs = slices.DeleteFunc(state.CheckedTagIDs, func(id int) bool {
			return id == int(data.TagID)
		})
	} else {
		state.CheckedTagIDs = append(state.CheckedTagIDs, int(data.TagID))
	}

	rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID))
	// After-Swap because otherwise command triggered by event are executed to early and
	// URL (HX-Current-URL) is not updated yet
	rw.Header().Set("HX-Trigger-After-Swap", event.FilterTagsChanged.String())

	// rw.AddRenderables(wx.NewSnackbarf("Tag filter toggled."))

	return nil
}
