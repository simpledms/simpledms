package browse

import (
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type UpdatePropertyFilterData struct {
	CurrentDirID string
	PropertyID   int64
}

type UpdatePropertyFilterFormData struct {
	UpdatePropertyFilterData
	Operator string
	Value    string // TODO typed? is string in URL...
}

type UpdatePropertyFilter struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUpdatePropertyFilter(infra *common.Infra, actions *Actions) *UpdatePropertyFilter {
	config := actionx.NewConfig(
		actions.Route("update-property-filter"),
		false,
	)
	return &UpdatePropertyFilter{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *UpdatePropertyFilter) Data(
	currentDirID string,
	propertyID int64,
	// operator, value string,
) *UpdatePropertyFilterData {
	return &UpdatePropertyFilterData{
		CurrentDirID: currentDirID,
		PropertyID:   propertyID,
		// Operator:     operator,
		// Value:        value,
	}
}

func (qq *UpdatePropertyFilter) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UpdatePropertyFilterFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[ListDirState](rw, req)

	var valuex PropertyFilterValue
	index := slices.IndexFunc(state.PropertiesFilterState.PropertyValues, func(valuex PropertyFilterValue) bool {
		return data.PropertyID == valuex.PropertyID
	})
	if index == -1 {
		valuex = PropertyFilterValue{
			PropertyID: data.PropertyID,
			Operator:   data.Operator, // TODO? is this case even possible or is it always initialized?
		}
	} else {
		valuex = state.PropertiesFilterState.PropertyValues[index]
	}

	valuex.Value = data.Value
	valuex.Operator = data.Operator

	state.PropertiesFilterState.PropertyValues[index] = valuex

	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)
	rw.AddRenderables(wx.NewSnackbarf("«%s» filter updated.", propertyx.Name))

	rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID))
	rw.Header().Set("HX-Trigger-After-Swap", event.PropertyFilterChanged.String())

	return nil
}
