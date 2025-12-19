package browse

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/action/tagging"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type SelectDirActions struct {
	// SelectDir *SelectDir // not factored out from MoveFile yet
	MakeDir *SelectDirMakeDir
}

type Actions struct {
	Common  *acommon.Actions
	Tagging *tagging.Actions

	ChangeDir  *ChangeDir
	ListDir    *ListDir
	MakeDir    *MakeDir
	DeleteFile *DeleteFile

	// SelectDir *SelectDirActions `actions:"select-dir"`

	FilePreview          *FilePreview
	FileDetailsSideSheet *FileDetailsSideSheet
	ShowFileInfo         *ShowFileInfo
	ShowFileTabs         *ShowFileTabs
	ShowFileSheet        *ShowFileSheet
	UploadFile           *UploadFile
	// UploadFileInFolderMode *UploadFileInFolderMode
	FileAttributes     *FileAttributes
	FileVersions       *FileVersions
	FileInfo           *FileInfo
	FileProperties     *FileProperties
	SelectDocumentType *SelectDocumentType
	SetFileProperty    *SetFileProperty

	// TODO rename to Rename and Move because they also work for folders?
	RenameFile *RenameFile
	MoveFile   *MoveFile

	FileListItem *FileListItem

	// SearchList *SearchList

	ListFilterTags           *ListFilterTags
	ListFilterProperties     *ListFilterProperties
	DocumentTypeFilter       *DocumentTypeFilter
	ToggleTagFilter          *ToggleTagFilter
	ToggleDocumentTypeFilter *ToggleDocumentTypeFilter
	TogglePropertyFilter     *TogglePropertyFilter
	DocumentTypeFilterDialog *DocumentTypeFilterDialog
	TagsFilterDialog         *TagsFilterDialog
	PropertiesFilterDialog   *PropertiesFilterDialog
	UpdatePropertyFilter     *UpdatePropertyFilter
	// ToggleFolderMode         *ToggleFolderMode

	FileUploadDialog *FileUploadDialog
	UnzipArchiveCmd  *UnzipArchiveCmd
}

func NewActions(infra *common.Infra, commonActions *acommon.Actions, taggingActions *tagging.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Common:  commonActions,
		Tagging: taggingActions,

		ChangeDir:  NewChangeDir(infra, actions),
		ListDir:    NewListDir(infra, actions),
		MakeDir:    NewMakeDir(infra, actions),
		DeleteFile: NewDeleteFile(infra, actions),

		// SelectDir: &SelectDirActions{
		// MakeDir: NewSelectDirMakeDir(infra, actions),
		// },

		FilePreview:          NewFilePreview(infra, actions),
		FileDetailsSideSheet: NewFileDetailsSideSheet(infra, actions),
		ShowFileInfo:         NewShowFileInfo(infra, actions),
		ShowFileTabs:         NewShowFileTabs(infra, actions),
		ShowFileSheet:        NewShowFileSheet(infra, actions),
		UploadFile:           NewUploadFile(infra, actions),
		// UploadFileInFolderMode: NewUploadFileInFolderMode(infra, actions),
		FileAttributes:     NewFileAttributes(infra, actions),
		FileVersions:       NewFileVersions(infra, actions),
		FileInfo:           NewFileInfo(infra, actions),
		FileProperties:     NewFileProperties(infra, actions),
		SelectDocumentType: NewSelectDocumentType(infra, actions),
		SetFileProperty:    NewSetFileProperty(infra, actions),

		RenameFile: NewRenameFile(infra, actions),
		MoveFile:   NewMoveFile(infra, actions),

		FileListItem: NewFileListItem(infra, actions),

		// SearchList: NewSearchList(infra, actions),

		ListFilterTags:           NewListFilterTags(infra, actions),
		ListFilterProperties:     NewListFilterProperties(infra, actions),
		DocumentTypeFilter:       NewDocumentTypeFilter(infra, actions),
		ToggleTagFilter:          NewToggleTagFilter(infra, actions),
		ToggleDocumentTypeFilter: NewToggleDocumentTypeFilter(infra, actions),
		TogglePropertyFilter:     NewTogglePropertyFilter(infra, actions),
		DocumentTypeFilterDialog: NewDocumentTypeFilterDialog(infra, actions),
		TagsFilterDialog:         NewTagsFilterDialog(infra, actions),
		PropertiesFilterDialog:   NewPropertiesFilterDialog(infra, actions),
		UpdatePropertyFilter:     NewUpdatePropertyFilter(infra, actions),
		// ToggleFolderMode:         NewToggleFolderMode(infra, actions),

		FileUploadDialog: NewFileUploadDialog(infra, actions),
		UnzipArchiveCmd:  NewUnzipArchiveCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.BrowseActionsRoute() + path
}
