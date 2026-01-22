package browse

import (
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type UpdatePropertyFilterCmdData struct {
	CurrentDirID string
	PropertyID   int64
}

type UpdatePropertyFilterCmdFormData struct {
	UpdatePropertyFilterCmdData
	Operator string
	Value    string // TODO typed? is string in URL...
}

type UpdatePropertyFilterCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUpdatePropertyFilterCmd(infra *common.Infra, actions *Actions) *UpdatePropertyFilterCmd {
	config := actionx.NewConfig(
		actions.Route("update-property-filter-cmd"),
		false,
	)
	return &UpdatePropertyFilterCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *UpdatePropertyFilterCmd) Data(
	currentDirID string,
	propertyID int64,
	// operator, value string,
) *UpdatePropertyFilterCmdData {
	return &UpdatePropertyFilterCmdData{
		CurrentDirID: currentDirID,
		PropertyID:   propertyID,
		// Operator:     operator,
		// Value:        value,
	}
}

func (qq *UpdatePropertyFilterCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UpdatePropertyFilterCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[ListDirPartialState](rw, req)

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
