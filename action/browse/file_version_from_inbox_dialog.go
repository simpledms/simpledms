package browse

import (
	"fmt"
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
	*actionx.Config
}

func NewFileVersionFromInboxDialog(infra *common.Infra, actions *Actions) *FileVersionFromInboxDialog {
	config := actionx.NewConfig(actions.Route("file-version-from-inbox-dialog"), true)
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

func (qq *FileVersionFromInboxDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
) *wx.Dialog {
	var formChildren []wx.IWidget

	formChildren = append(formChildren,
		&wx.Checkbox{
			Name:       "ConfirmWarning",
			Label:      wx.T("I understand that the inbox file's metadata (document type, tags, fields) will be lost when merged."),
			IsRequired: true,
		},
	)

	formChildren = append(formChildren,
		wx.NewFormFields(ctx, &FileVersionFromInboxDialogFormData{
			TargetFileID: data.TargetFileID,
		}),
		&wx.Search{
			Widget: wx.Widget[wx.Search]{
				ID: qq.searchID(),
			},
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:    qq.actions.FileVersionFromInboxListPartial.Endpoint(),
				HxTarget:  "#" + qq.listID(),
				HxSelect:  "#" + qq.listID(),
				HxSwap:    "outerHTML",
				HxTrigger: fmt.Sprintf("input from:#%s delay:150ms", qq.searchID()),
				HxInclude: "#" + qq.searchID() + ", #" + qq.formID(),
			},
			Name:           "SearchQuery",
			Value:          data.SearchQuery,
			SupportingText: wx.T("Search inbox files"),
			Autofocus:      true,
		},
		qq.actions.FileVersionFromInboxListPartial.listWrapper(ctx, data, files),
	)

	content := &wx.Container{
		GapY: true,
		Child: &wx.Form{
			Widget: wx.Widget[wx.Form]{
				ID: qq.formID(),
			},
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.FileVersionFromInboxCmd.Endpoint(),
				HxSwap: "none",
			},
			Children: formChildren,
		},
	}

	return autil.WrapWidgetWithID(
		wx.T("Add new version from inbox"),
		wx.T("Add"),
		content,
		actionx.ResponseWrapperDialog,
		wx.DialogLayoutStable,
		qq.dialogID(),
		qq.formID(),
	).(*wx.Dialog)
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
