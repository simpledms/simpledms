package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ToggleDocumentTypeFilterData struct {
	CurrentDirID   string
	DocumentTypeID int64
}

type ToggleDocumentTypeFilter struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleDocumentTypeFilter(infra *common.Infra, actions *Actions) *ToggleDocumentTypeFilter {
	config := actionx.NewConfig(
		actions.Route("toggle-document-type-filter"),
		true,
	)
	return &ToggleDocumentTypeFilter{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ToggleDocumentTypeFilter) Data(currentDirID string, documentTypeID int64) *ToggleDocumentTypeFilterData {
	return &ToggleDocumentTypeFilterData{
		CurrentDirID:   currentDirID,
		DocumentTypeID: documentTypeID,
	}
}

func (qq *ToggleDocumentTypeFilter) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ToggleDocumentTypeFilterData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirState](rw, req)

	if state.DocumentTypeID == data.DocumentTypeID {
		state.DocumentTypeID = 0
	} else {
		state.DocumentTypeID = data.DocumentTypeID
	}

	rw.Header().Set("HX-Replace-Url", route.BrowseWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID))
	// After-Swap because otherwise command triggered by event are executed to early and
	// URL (HX-Current-URL) is not updated yet
	rw.Header().Set("HX-Trigger-After-Swap", event.DocumentTypeFilterChanged.String())

	return nil
}
