package browse

import (
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ResetFiltersCmdData struct {
	CurrentDirID string
}

type ResetFiltersCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewResetFiltersCmd(infra *common.Infra, actions *Actions) *ResetFiltersCmd {
	config := actionx.NewConfig(
		actions.Route("reset-filters-cmd"),
		true,
	)
	return &ResetFiltersCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ResetFiltersCmd) Data(currentDirID string) *ResetFiltersCmdData {
	return &ResetFiltersCmdData{
		CurrentDirID: currentDirID,
	}
}

// Handler resets only the filters of the side sheet; search, sorting, and the open side sheet
// are preserved.
func (qq *ResetFiltersCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ResetFiltersCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirPartialState](rw, req)
	state.resetFilters()

	rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID))
	// After-Swap because otherwise command triggered by event are executed to early and
	// URL (HX-Current-URL) is not updated yet
	rw.Header().Set("HX-Trigger-After-Swap", strings.Join([]string{
		event.FilterTagsChanged.String(),
		event.DocumentTypeFilterChanged.String(),
		event.PropertyFilterChanged.String(),
	}, ", "))
	rw.AddRenderables(widget.NewSnackbarf("Filters reset."))

	return nil
}
