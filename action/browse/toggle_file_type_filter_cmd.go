package browse

import (
	"net/url"
	"slices"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ToggleFileTypeFilterCmd struct {
	*actionx.Config
}

func NewToggleFileTypeFilterCmd(actions *Actions) *ToggleFileTypeFilterCmd {
	return &ToggleFileTypeFilterCmd{
		Config: actionx.NewConfig(actions.Route("toggle-file-type-filter-cmd"), true),
	}
}

func (qq *ToggleFileTypeFilterCmd) Data(
	currentDirID, fileType string,
) *ToggleFileTypeFilterCmdData {
	return &ToggleFileTypeFilterCmdData{
		CurrentDirID: currentDirID,
		FileType:     fileType,
	}
}

func (qq *ToggleFileTypeFilterCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[ToggleFileTypeFilterCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirPartialState](rw, req)
	if slices.Contains(state.FileTypes, data.FileType) {
		state.FileTypes = slices.DeleteFunc(state.FileTypes, func(value string) bool {
			return value == data.FileType
		})
	} else {
		state.FileTypes = append(state.FileTypes, data.FileType)
	}

	tenantID := ctx.TenantCtx().TenantID
	spaceID := ctx.SpaceCtx().SpaceID
	location := route.BrowseWithState(state)(tenantID, spaceID, data.CurrentDirID)
	if current, err := url.Parse(req.Header.Get("HX-Current-URL")); err == nil {
		if _, selectedFileID, found := strings.Cut(current.Path, "/file/"); found {
			location = route.BrowseFileWithState(state)(
				tenantID, spaceID, data.CurrentDirID, selectedFileID,
			)
		}
	}
	rw.Header().Set("HX-Replace-Url", location)
	// Refresh islands after the URL contains the new selection.
	rw.Header().Set("HX-Trigger-After-Swap", event.FileTypeFilterChanged.String())
	return nil
}
