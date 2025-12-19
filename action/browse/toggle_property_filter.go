package browse

import (
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type TogglePropertyFilterData struct {
	CurrentDirID string
	PropertyID   int64
}

type TogglePropertyFilter struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewTogglePropertyFilter(infra *common.Infra, actions *Actions) *TogglePropertyFilter {
	config := actionx.NewConfig(
		actions.Route("toggle-property-filter"),
		true,
	)
	return &TogglePropertyFilter{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *TogglePropertyFilter) Data(currentDirID string, propertyID int64) *TogglePropertyFilterData {
	return &TogglePropertyFilterData{
		CurrentDirID: currentDirID,
		PropertyID:   propertyID,
	}
}

func (qq *TogglePropertyFilter) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[TogglePropertyFilterData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirState](rw, req)

	if slices.ContainsFunc(state.PropertyValues, func(value PropertyFilterValue) bool {
		return value.PropertyID == data.PropertyID
	}) {
		slices.DeleteFunc(state.PropertyValues, func(value PropertyFilterValue) bool {
			return value.PropertyID == data.PropertyID
		})
	} else {
		state.PropertyValues = append(state.PropertyValues, PropertyFilterValue{
			PropertyID: data.PropertyID,
		})
	}

	rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID))
	// After-Swap because otherwise command triggered by event are executed too early and
	// URL (HX-Current-URL) is not updated yet
	rw.Header().Set("HX-Trigger-After-Swap", event.PropertyFilterChanged.String())

	return nil
}
