package browse

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	previewconversion "github.com/simpledms/simpledms/db/enttenant/previewconversion"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/internal/gotenberg"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	previewmodel "github.com/simpledms/simpledms/model/tenant/previewconversion"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePreviewPartialData struct {
	CurrentDirID string
	FileID       string
}

type FilePreviewPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

type FilePreviewPartialState struct {
	ListDirPartialState
	ActiveTab  string `url:"tab,omitempty"`
	PreviewTab string `url:"preview_tab,omitempty"`
}

func NewFilePreviewPartial(infra *common.Infra, actions *Actions) *FilePreviewPartial {
	return &FilePreviewPartial{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("file-preview-partial"),
			true,
		),
	}
}

const previewStatusChangedTrigger = "previewStatusChanged from:body"

func (qq *FilePreviewPartial) PreviewWidget(
	ctx ctxx.Context,
	filex *filemodel.File,
	source *enttenant.StoredFile,
	versionNumber string,
	currentDirID string,
	activeTab string,
) (renderable.Renderable, bool, error) {
	classification, eligible := previewmodel.Classify(source.MimeType, source.Filename, filex.Data.IsDirectory)
	if !eligible {
		return &widget.FilePreview{
			FileURL:  qq.originalInlineURL(ctx, filex, versionNumber),
			Filename: source.Filename,
			MimeType: source.MimeType,
		}, false, nil
	}

	configured := gotenberg.IsValidGotenbergURL(qq.infra.SystemConfig().GotenbergURL())
	conversion, preview, err := qq.previewConversion(ctx, source.ID)
	if err != nil {
		return nil, true, err
	}
	ready := conversion != nil && conversion.Status == previewmodel.Ready && preview != nil && preview.CopiedToFinalDestinationAt != nil
	pending := configured && !ready && (conversion == nil || conversion.Status == previewmodel.Pending || conversion.Status == previewmodel.Processing)
	failed := configured && conversion != nil && conversion.Status == previewmodel.Failed

	activeTab = strings.ToLower(activeTab)
	hasExplicitTab := activeTab == "preview" || activeTab == "original"
	if activeTab != "preview" && activeTab != "original" {
		activeTab = "preview"
	}

	tabsID := "browsePreviewTabs"
	if versionNumber != "" {
		tabsID += "Version" + versionNumber
	}
	statusData := qq.actions.FilePreviewStatusPartial.Data(
		currentDirID,
		filex.Data.PublicID.String(),
		versionNumber,
		"",
	)
	if hasExplicitTab {
		statusData.PreviewTab = activeTab
	}

	tabAttrs := func(tab string) widget.HTMXAttrs {
		tabData := qq.actions.FilePreviewStatusPartial.Data(
			currentDirID,
			filex.Data.PublicID.String(),
			versionNumber,
			tab,
		)
		tabData.PushURL = currentDirID == "" && versionNumber == ""
		return widget.HTMXAttrs{
			HxPost:   qq.actions.FilePreviewStatusPartial.Endpoint(),
			HxVals:   util.JSON(tabData),
			HxTarget: "#" + tabsID,
			HxSwap:   "outerHTML",
			HxSync:   "#" + tabsID + ":replace",
		}
	}

	previewTab := &widget.Tab{
		Label:     widget.T("Preview"),
		HTMXAttrs: tabAttrs("preview"),
	}
	originalTab := &widget.Tab{
		Label:     widget.T("Original"),
		HTMXAttrs: tabAttrs("original"),
	}

	var content widget.IWidget
	if activeTab == "preview" {
		switch {
		case ready:
			content = &widget.Column{
				GapYSize:         widget.Gap2,
				NoOverflowHidden: true,
				Children: []widget.IWidget{
					widget.NewToolbar(
						widget.T("Preview").String(ctx),
						&widget.Link{
							Href:      qq.previewDownloadURL(ctx, filex, versionNumber),
							Filename:  preview.Filename,
							IsNoColor: true,
							Child: &widget.IconButton{
								Icon:    "download",
								Tooltip: widget.T("Download PDF"),
							},
						},
					),
					&widget.FilePreview{
						FileURL:  qq.previewInlineURL(ctx, filex, versionNumber),
						Filename: preview.Filename,
						MimeType: "application/pdf",
					},
				},
			}
		case failed:
			content = &widget.Column{
				GapYSize: widget.Gap2,
				Children: []widget.IWidget{
					qq.previewStatusMessage(widget.T("PDF preview could not be generated.")),
					&widget.Button{
						Label: widget.T("Retry PDF generation"),
						HTMXAttrs: widget.HTMXAttrs{
							HxPost:   qq.actions.RetryPDFPreviewCmd.Endpoint(),
							HxVals:   util.JSON(qq.actions.RetryPDFPreviewCmd.Data(currentDirID, filex.Data.PublicID.String(), versionNumber)),
							HxSwap:   "none",
							HxTarget: "#" + tabsID,
						},
					},
				},
			}
		default:
			content = qq.previewStateMessage(configured, pending, failed)
		}
	} else {
		content = &widget.Column{
			GapYSize:         widget.Gap2,
			NoOverflowHidden: true,
			Children: []widget.IWidget{
				widget.NewToolbar(
					widget.T("Original").String(ctx),
					&widget.Link{
						Href:      qq.originalDownloadURL(ctx, filex, versionNumber),
						Filename:  source.Filename,
						IsNoColor: true,
						Child: &widget.IconButton{
							Icon:    "download",
							Tooltip: widget.T("Download"),
						},
					},
				),
				&widget.FilePreview{
					FileURL: qq.originalPreviewURL(
						ctx,
						filex,
						source,
						classification.Family,
						versionNumber,
					),
					Filename:         source.Filename,
					MimeType:         qq.originalPreviewMIME(source.MimeType, classification.Family),
					HideDownloadLink: true,
				},
			},
		}
	}

	pollTrigger := ""
	if pending {
		pollTrigger = "every 5s," + previewStatusChangedTrigger
	} else if failed {
		pollTrigger = previewStatusChangedTrigger
	}

	return &widget.Column{
		Widget: widget.Widget[widget.Column]{ID: tabsID},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:      qq.actions.FilePreviewStatusPartial.Endpoint(),
			HxTrigger:   pollTrigger,
			HxVals:      util.JSON(statusData),
			HxTarget:    "#" + tabsID,
			HxSwap:      "outerHTML",
			HxPushURL:   "false",
			HxIndicator: "closest .js-preview-status",
		},
		Children: &widget.TabBar{
			Widget:           widget.Widget[widget.TabBar]{ID: tabsID + "Bar"},
			ActiveTab:        activeTab,
			IsFlowing:        true,
			Tabs:             []*widget.Tab{previewTab, originalTab},
			ActiveTabContent: content,
		},
	}, true, nil
}

func (qq *FilePreviewPartial) previewConversion(
	ctx ctxx.Context,
	sourceStoredFileID int64,
) (*enttenant.PreviewConversion, *enttenant.StoredFile, error) {
	conversion, err := ctx.TenantCtx().TTx.PreviewConversion.Query().
		Where(previewconversion.SourceStoredFileID(sourceStoredFileID)).
		Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if conversion.PreviewStoredFileID == nil {
		return conversion, nil, nil
	}
	previewContext := tenantprivacy.DecisionContext(ctx, tenantprivacy.Allow)
	preview, err := ctx.TenantCtx().TTx.StoredFile.Get(previewContext, *conversion.PreviewStoredFileID)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return conversion, nil, nil
		}
		return nil, nil, err
	}
	return conversion, preview, nil
}

func (qq *FilePreviewPartial) previewStateMessage(configured, pending, failed bool) widget.IWidget {
	if !configured {
		return qq.previewStatusMessage(widget.T("PDF preview is unavailable because Gotenberg is not configured."))
	}
	if pending {
		return qq.previewStatusMessage(widget.T("PDF preview is being generated. Please wait a moment; the page will refresh automatically."))
	}
	if failed {
		return qq.previewStatusMessage(widget.T("PDF preview could not be generated."))
	}
	return qq.previewStatusMessage(widget.T("PDF preview is not available."))
}

func (qq *FilePreviewPartial) previewStatusMessage(text *widget.Text) *widget.Paragraph {
	message := widget.NewParagraph(text)
	message.Class = "body-lg text-on-surface p-4"
	return message
}

func (qq *FilePreviewPartial) originalPreviewURL(
	ctx ctxx.Context,
	filex *filemodel.File,
	source *enttenant.StoredFile,
	family previewmodel.Family,
	versionNumber string,
) string {
	if family == previewmodel.FamilyHTML {
		if versionNumber == "" {
			return route2.OriginalSource(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String())
		}
		return route2.OriginalSourceWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), versionNumber)
	}
	return qq.originalInlineURL(ctx, filex, versionNumber)
}

func (qq *FilePreviewPartial) originalInlineURL(ctx ctxx.Context, filex *filemodel.File, versionNumber string) string {
	if versionNumber == "" {
		return route2.DownloadInline(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String())
	}
	return route2.DownloadInlineWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), versionNumber)
}

func (qq *FilePreviewPartial) previewInlineURL(ctx ctxx.Context, filex *filemodel.File, versionNumber string) string {
	if versionNumber == "" {
		return route2.PreviewPDF(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String())
	}
	return route2.PreviewPDFWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), versionNumber)
}

func (qq *FilePreviewPartial) previewDownloadURL(ctx ctxx.Context, filex *filemodel.File, versionNumber string) string {
	if versionNumber == "" {
		return route2.PreviewPDFDownload(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String())
	}
	return route2.PreviewPDFDownloadWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), versionNumber)
}

func (qq *FilePreviewPartial) originalDownloadURL(ctx ctxx.Context, filex *filemodel.File, versionNumber string) string {
	if versionNumber == "" {
		return route2.Download(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String())
	}
	return route2.DownloadWithVersion(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String(), versionNumber)
}

func (qq *FilePreviewPartial) originalPreviewMIME(mimeType string, family previewmodel.Family) string {
	if family == previewmodel.FamilyHTML {
		return "text/plain"
	}
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

func (qq *FilePreviewPartial) versionSource(ctx ctxx.Context, filex *filemodel.File, versionNumber string) (*enttenant.StoredFile, error) {
	if versionNumber == "" {
		return filex.CurrentVersion(ctx).Data, nil
	}
	versionInt, err := strconv.Atoi(versionNumber)
	if err != nil {
		return nil, err
	}
	version, err := filex.Data.QueryFileVersions().
		Where(fileversion.VersionNumber(versionInt)).
		WithStoredFile().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return version.Edges.StoredFile, nil
}

func (qq *FilePreviewPartial) Data(currentDirID, fileID string) *FilePreviewPartialData {
	return &FilePreviewPartialData{
		CurrentDirID: currentDirID,
		FileID:       fileID,
	}
}

func (qq *FilePreviewPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilePreviewPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[FilePreviewPartialState](rw, req)
	rw.Header().Set("HX-Push-Url", route2.BrowseFileWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID, data.FileID))

	// filex := ctx.TenantCtx().TTx.File.GetX(ctx, data.FileID)
	dirx := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	viewx, err := qq.Widget(ctx, state, dirx, filex)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "rendering failed")
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
	return nil
}

func (qq *FilePreviewPartial) Widget(
	ctx ctxx.Context,
	state *FilePreviewPartialState,
	dirx *filemodel.File,
	filex *filemodel.File,
) (*widget.DetailsWithSheet, error) {
	// TODO action.ShowFileData or primitive types?
	//		is partial bound to action?
	//
	// TODO stream file (also necessary for download)

	// FIXME update on changes...
	// soft delete filter is not applied via TagAssignment
	// tagsCount := qq.infra.Client().File.GetX(ctx, filepathxx.FileID).QueryTags().CountX(ctx)

	fileDetailsSideSheet := qq.actions.FileDetailsSideSheetPartial.Widget(
		ctx,
		qq.actions.FileDetailsSideSheetPartial.Data(
			dirx.Data.PublicID.String(),
			filex.Data.PublicID.String(),
		),
		state,
	)

	currentVersion := filex.CurrentVersion(ctx)
	preview, hasPreviewTabs, err := qq.PreviewWidget(
		ctx,
		filex,
		currentVersion.Data,
		"",
		dirx.Data.PublicID.String(),
		state.PreviewTab,
	)
	if err != nil {
		return nil, err
	}

	return &widget.DetailsWithSheet{
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: event.FileUploaded.Handler(),
			HxPost:    qq.Endpoint(),
			HxVals:    util.JSON(qq.Data(dirx.Data.PublicID.String(), filex.Data.PublicID.String())),
			HxTarget:  "#details",
			HxSwap:    "outerHTML",
		},
		AppBar: qq.appBar(
			ctx,
			dirx.Data.PublicID.String(),
			widget.Tu(filex.FilenameInApp(ctx, true)),
			filex,
			filex.Filename(ctx),
			hasPreviewTabs,
		),
		Child: &widget.Column{
			Children: preview,
		},
		SideSheet: fileDetailsSideSheet,
	}, nil
}

func (qq *FilePreviewPartial) appBar(
	ctx ctxx.Context,
	dirID string,
	title *widget.Text,
	filex *filemodel.File,
	filename string,
	hasPreviewTabs bool,
) *widget.AppBar {
	actions := []widget.IWidget{
		&widget.IconButton{
			// TODO other icon if already open or hide...
			Icon:    "description", // right_panel_open, clarify, tune, description, info, ...?
			Tooltip: widget.T("Show details"),
			HTMXAttrs: widget.HTMXAttrs{
				DialogID: qq.actions.FileDetailsSideSheetPartial.ID(),
			},
		},
	}
	if !hasPreviewTabs {
		actions = append(actions, &widget.Link{
			Href: route2.Download(
				ctx.TenantCtx().TenantID,
				ctx.SpaceCtx().SpaceID,
				filex.Data.PublicID.String(),
			),
			IsNoColor: true,
			Filename:  filename,
			Child: &widget.IconButton{
				Icon:    "download",
				Tooltip: widget.T("Download"),
			},
		})
	}

	return &widget.AppBar{
		Leading: &widget.IconButton{
			Icon:    "close",
			Tooltip: widget.T("Close preview"),
			// TODO use link instead?
			HTMXAttrs: widget.HTMXAttrs{
				HxGet:     route2.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, dirID),
				HxOn:      event.DetailsClosed.HxOn("click"),
				HxHeaders: autil.CloseDetailsHeader(),
			},
		},
		Title: &widget.AppBarTitle{
			Text: title,
		},
		Actions: actions,
		/*
			&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{
						{
							LeadingIcon:          "download",
							Label:                wx.T("Download"),
							DownloadLinkURL:      route.Download(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
							DownloadLinkFilename: filex.Filename(ctx),
						},
					},
				},
			},
		*/
	}
}
