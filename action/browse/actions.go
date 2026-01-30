package browse

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/action/tagging"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type SelectDirActions struct {
	// SelectDirPartial *SelectDirPartial // not factored out from MoveFileCmd yet
	MakeDirCmd *SelectDirMakeDirCmd
}

type Actions struct {
	Common  *acommon.Actions
	Tagging *tagging.Actions

	BrowsePage              *BrowsePage
	BrowseWithSelectionPage *BrowseWithSelectionPage

	ChangeDirPartial *ChangeDirCmd
	ListDirPartial   *ListDirPartial
	MakeDirCmd       *MakeDirCmd
	DeleteFile       *DeleteFile

	// SelectDirPartial *SelectDirActions `actions:"select-dir"`

	FilePreviewPartial          *FilePreviewPartial
	FileDetailsSideSheetPartial *FileDetailsSideSheetPartial
	FileTabsPartial             *FileTabsPartial
	FileSheetPartial            *FileSheetPartial
	UploadFileCmd               *UploadFileCmd
	UploadFileVersionCmd        *UploadFileVersionCmd
	// UploadFileInFolderMode *UploadFileInFolderMode
	FileAttributesPartial      *FileAttributesPartial
	FileVersionsPartial        *FileVersionsPartial
	FileInfoPartial            *FileInfoPartial
	FilePropertiesPartial      *FilePropertiesPartial
	AddFilePropertyCmd         *AddFilePropertyCmd
	AddFilePropertyValueDialog *AddFilePropertyValueDialog
	AddFilePropertyValueCmd    *AddFilePropertyValueCmd
	RemoveFilePropertyCmd      *RemoveFilePropertyCmd
	SelectDocumentTypePartial  *SelectDocumentTypeCmd
	SetFilePropertyCmd         *SetFilePropertyCmd

	// TODO rename to Rename and Move because they also work for folders?
	RenameFileCmd *RenameFileCmd
	MoveFileCmd   *MoveFileCmd

	FileListItemPartial *FileListItemPartial

	// SearchList *SearchList

	ListFilterTagsPartial           *ListFilterTagsPartial
	ListFilterPropertiesPartial     *ListFilterPropertiesPartial
	DocumentTypeFilterPartial       *DocumentTypeFilterPartial
	ToggleTagFilterCmd              *ToggleTagFilterCmd
	ToggleDocumentTypeFilterCmd     *ToggleDocumentTypeFilterCmd
	TogglePropertyFilterCmd         *TogglePropertyFilterCmd
	DocumentTypeFilterDialogPartial *DocumentTypeFilterDialog
	TagsFilterDialogPartial         *TagsFilterDialog
	PropertiesFilterDialogPartial   *PropertiesFilterDialog
	UpdatePropertyFilterCmd         *UpdatePropertyFilterCmd
	// ToggleFolderMode         *ToggleFolderMode

	FileUploadDialogPartial         *FileUploadDialog
	FileVersionUploadDialogPartial  *FileVersionUploadDialog
	FileVersionPreviewDialogPartial *FileVersionPreviewDialog
	UnzipArchiveCmd                 *UnzipArchiveCmd
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions, taggingActions *tagging.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common:  commonActions,
		Tagging: taggingActions,

		BrowsePage:              NewBrowsePage(infra, actions),
		BrowseWithSelectionPage: NewBrowseWithSelectionPage(infra, actions),

		ChangeDirPartial: NewChangeDirCmd(infra, actions),
		ListDirPartial:   NewListDirPartial(infra, actions),
		MakeDirCmd:       NewMakeDirCmd(infra, actions),
		DeleteFile:       NewDeleteFile(infra, actions),

		// SelectDirPartial: &SelectDirActions{
		// MakeDirCmd: NewSelectDirMakeDirCmd(infra, actions),
		// },

		FilePreviewPartial:          NewFilePreviewPartial(infra, actions),
		FileDetailsSideSheetPartial: NewFileDetailsSideSheetPartial(infra, actions),
		FileTabsPartial:             NewFileTabsPartial(infra, actions),
		FileSheetPartial:            NewFileSheetPartial(infra, actions),
		UploadFileCmd:               NewUploadFileCmd(infra, actions),
		UploadFileVersionCmd:        NewUploadFileVersionCmd(infra, actions),
		// UploadFileInFolderMode: NewUploadFileInFolderMode(infra, actions),
		FileAttributesPartial:      NewFileAttributesPartial(infra, actions),
		FileVersionsPartial:        NewFileVersionsPartial(infra, actions),
		FileInfoPartial:            NewFileInfoPartial(infra, actions),
		FilePropertiesPartial:      NewFilePropertiesPartial(infra, actions),
		AddFilePropertyCmd:         NewAddFilePropertyCmd(infra, actions),
		AddFilePropertyValueDialog: NewAddFilePropertyValueDialog(infra, actions),
		AddFilePropertyValueCmd:    NewAddFilePropertyValueCmd(infra, actions),
		RemoveFilePropertyCmd:      NewRemoveFilePropertyCmd(infra, actions),
		SelectDocumentTypePartial:  NewSelectDocumentTypeCmd(infra, actions),
		SetFilePropertyCmd:         NewSetFilePropertyCmd(infra, actions),

		RenameFileCmd: NewRenameFileCmd(infra, actions),
		MoveFileCmd:   NewMoveFileCmd(infra, actions),

		FileListItemPartial: NewFileListItemPartial(infra, actions),

		// SearchList: NewSearchList(infra, actions),

		ListFilterTagsPartial:           NewListFilterTagsPartial(infra, actions),
		ListFilterPropertiesPartial:     NewListFilterPropertiesPartial(infra, actions),
		DocumentTypeFilterPartial:       NewDocumentTypeFilterPartial(infra, actions),
		ToggleTagFilterCmd:              NewToggleTagFilterCmd(infra, actions),
		ToggleDocumentTypeFilterCmd:     NewToggleDocumentTypeFilterCmd(infra, actions),
		TogglePropertyFilterCmd:         NewTogglePropertyFilterCmd(infra, actions),
		DocumentTypeFilterDialogPartial: NewDocumentTypeFilterDialog(infra, actions),
		TagsFilterDialogPartial:         NewTagsFilterDialog(infra, actions),
		PropertiesFilterDialogPartial:   NewPropertiesFilterDialog(infra, actions),
		UpdatePropertyFilterCmd:         NewUpdatePropertyFilterCmd(infra, actions),
		// ToggleFolderMode:         NewToggleFolderMode(infra, actions),

		FileUploadDialogPartial:         NewFileUploadDialog(infra, actions),
		FileVersionUploadDialogPartial:  NewFileVersionUploadDialog(infra, actions),
		FileVersionPreviewDialogPartial: NewFileVersionPreviewDialog(infra, actions),
		UnzipArchiveCmd:                 NewUnzipArchiveCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.BrowseActionsRoute() + path
}
