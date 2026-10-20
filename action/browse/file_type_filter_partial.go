package browse

import (
	"encoding/base64"
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileTypeFilterPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileTypeFilterPartial(infra *common.Infra, actions *Actions) *FileTypeFilterPartial {
	return &FileTypeFilterPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("file-type-filter-partial"), true),
	}
}

func (qq *FileTypeFilterPartial) Data(currentDirID string) *FileTypeFilterPartialData {
	return &FileTypeFilterPartialData{
		CurrentDirID: currentDirID,
	}
}

func (qq *FileTypeFilterPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[FileTypeFilterPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirPartialState](rw, req)
	content, err := qq.Widget(ctx, data.CurrentDirID, state)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(rw, ctx, content)
}

func (qq *FileTypeFilterPartial) Widget(
	ctx ctxx.Context,
	currentDirID string,
	state *ListDirPartialState,
) (*widget.Container, error) {
	types, err := filemodel.NewFileTypeFilter().Available(ctx)
	if err != nil {
		return nil, err
	}
	// Keep selected types removable if their last file was deleted or renamed.
	for _, extension := range state.FileTypes {
		if !slices.Contains(types, extension) {
			types = append(types, extension)
		}
	}
	slices.Sort(types)
	children := make([]widget.IWidget, 0, len(types))
	for _, extension := range types {
		label := widget.Tu(extension)
		if extension == filemodel.FileTypeWithoutExtension {
			label = widget.T("No extension")
		}
		children = append(children, &widget.FilterChip{
			Widget: widget.Widget[widget.FilterChip]{
				// Stable nodes keep queued changes alive across panel morphs.
				ID: qq.ID() + "-" + base64.RawURLEncoding.EncodeToString([]byte(extension)),
			},
			Type:      widget.FilterChipTypeCheckbox,
			Name:      "FileTypes",
			Value:     extension,
			Label:     label,
			IsChecked: slices.Contains(state.FileTypes, extension),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.actions.ToggleFileTypeFilterCmd.Endpoint(),
				HxVals:    util.JSON(qq.actions.ToggleFileTypeFilterCmd.Data(currentDirID, extension)),
				HxTrigger: "change",
				HxSwap:    "none",
				// The app disables HTMX inheritance; chips must opt into the panel's queue.
				HxSync: "#" + qq.ID() + ":queue all",
			},
		})
	}
	if len(children) == 0 {
		children = append(children, &widget.EmptyState{
			Headline: widget.T("No file types available yet."),
		})
	}
	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.ID(),
		},
		Gap:   true,
		Child: children,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID)),
			HxTarget: "#" + qq.ID(),
			HxSwap:   "outerHTML",
			// Commands and refreshes share one queue so refreshes cannot discard pending changes.
			HxSync: "#" + qq.ID() + ":queue all",
			HxTrigger: event.HxTrigger(
				event.FileTypeFilterChanged,
				event.FileUploaded,
				event.FileUpdated,
				event.FileDeleted,
				event.FileRestored,
				event.FileMoved,
				event.InboxChanged,
				event.ZIPArchiveUnzipped,
			),
		},
	}, nil
}

func (qq *FileTypeFilterPartial) ID() string {
	return "fileTypeFilter"
}
