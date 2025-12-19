package inbox

import (
	"github.com/simpledms/simpledms/app/simpledms/action/browse"
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
)

type Actions struct {
	Common *acommon.Actions
	Browse *browse.Actions // TODO get rid of?

	Page *Page

	ListFiles    *ListFiles
	FileListItem *FileListItem
	FileMetadata *FileMetadata

	ListInboxAssignmentSuggestions *ListInboxAssignmentSuggestions
	AssignmentDirectoryListItem    *AssignmentDirectoryListItem
	AssignFile                     *AssignFile

	ShowFile     *ShowFile
	ShowFileTabs *ShowFileTabs
	MoveFile     *MoveFile
	UploadFile   *UploadFile
	MarkAsDone   *MarkAsDone
}

func NewActions(
	infra *common.Infra,
	commonActions *acommon.Actions,
	browseActions *browse.Actions,
) *Actions {
	actions := new(Actions)

	/*db := infra.UnsafeDB()
	ctx := context.Background()*/

	*actions = Actions{
		Common: commonActions,
		Browse: browseActions,

		Page: NewPage(infra, actions),

		ListFiles:                      NewListFiles(infra, actions),
		FileListItem:                   NewFileListItem(infra, actions),
		ListInboxAssignmentSuggestions: NewListInboxAssignmentSuggestions(infra, actions),
		AssignmentDirectoryListItem:    NewAssignmentDirectoryListItem(infra, actions),
		AssignFile:                     NewAssignFile(infra, actions),

		ShowFile:     NewShowFile(infra, actions),
		ShowFileTabs: NewShowFileTabs(infra, actions),
		MoveFile:     NewMoveFile(infra, actions),
		UploadFile:   NewUploadFile(infra, actions),

		MarkAsDone: NewMarkAsDone(infra, actions),
	}

	// uses actions in constructor, thus outside
	actions.FileMetadata = NewFileMetadata(infra, actions)

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.InboxActionsRoute() + path
}
