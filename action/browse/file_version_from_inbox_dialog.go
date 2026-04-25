package browse

import (
	"fmt"
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
)

type FileVersionFromInboxDialogData struct {
	TargetFileID string
	SourceFileID string
	SearchQuery  string
}

type FileVersionFromInboxDialogFormData struct {
	TargetFileID string `form_attr_type:"hidden"`
}

type FileVersionFromInboxDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewFileVersionFromInboxDialog(infra *common.Infra, actions *Actions) *FileVersionFromInboxDialog {
	config := actionx2.NewConfig(actions.Route("file-version-from-inbox-dialog"), true)
	return &FileVersionFromInboxDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionFromInboxDialog) Data(targetFileID, sourceFileID, searchQuery string) *FileVersionFromInboxDialogData {
	return &FileVersionFromInboxDialogData{
		TargetFileID: targetFileID,
		SourceFileID: sourceFileID,
		SearchQuery:  searchQuery,
	}
}

func (qq *FileVersionFromInboxDialog) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionFromInboxDialogData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	files, err := qq.actions.FileVersionFromInboxListPartial.listFiles(ctx, data)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data, files),
	)
}

func (qq *FileVersionFromInboxDialog) Widget(
	ctx ctxx.Context,
	data *FileVersionFromInboxDialogData,
	files []*enttenant.File,
) *widget.Dialog {
	var formChildren []widget.IWidget

	formChildren = append(formChildren,
		&widget.Checkbox{
			Name:       "ConfirmWarning",
			Label:      widget.T("I understand that the inbox file's metadata (document type, tags, fields) will be lost when merged."),
			IsRequired: true,
		},
	)

	formChildren = append(formChildren,
		widget.NewFormFields(ctx, &FileVersionFromInboxDialogFormData{
			TargetFileID: data.TargetFileID,
		}),
		&widget.Search{
			Widget: widget.Widget[widget.Search]{
				ID: qq.searchID(),
			},
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.actions.FileVersionFromInboxListPartial.Endpoint(),
				HxTarget:  "#" + qq.listID(),
				HxSelect:  "#" + qq.listID(),
				HxSwap:    "outerHTML",
				HxTrigger: fmt.Sprintf("input from:#%s delay:150ms", qq.searchID()),
				HxInclude: "#" + qq.searchID() + ", #" + qq.formID(),
			},
			Name:           "SearchQuery",
			Value:          data.SearchQuery,
			SupportingText: widget.T("Search inbox files"),
			Autofocus:      true,
		},
		qq.actions.FileVersionFromInboxListPartial.listWrapper(ctx, data, files),
	)

	content := &widget.Container{
		GapY: true,
		Child: &widget.Form{
			Widget: widget.Widget[widget.Form]{
				ID: qq.formID(),
			},
			HTMXAttrs: widget.HTMXAttrs{
				HxPost: qq.actions.FileVersionFromInboxCmd.Endpoint(),
				HxSwap: "none",
			},
			Children: formChildren,
		},
	}

	return autil.WrapWidgetWithID(
		widget.T("Add new version from inbox"),
		widget.T("Add"),
		content,
		actionx2.ResponseWrapperDialog,
		widget.DialogLayoutStable,
		qq.dialogID(),
		qq.formID(),
	).(*widget.Dialog)
}

func (qq *FileVersionFromInboxDialog) dialogID() string {
	return "fileVersionFromInboxDialog"
}

func (qq *FileVersionFromInboxDialog) formID() string {
	return "fileVersionFromInboxForm"
}

func (qq *FileVersionFromInboxDialog) searchID() string {
	return "fileVersionFromInboxSearch"
}

func (qq *FileVersionFromInboxDialog) listID() string {
	return "fileVersionFromInboxList"
}
