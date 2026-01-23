package inbox

import (
	"github.com/simpledms/simpledms/action/browse"
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	Common *acommon.Actions
	Browse *browse.Actions // TODO get rid of?

	InboxRootPage          *InboxRootPage
	InboxWithSelectionPage *InboxWithSelectionPage

	InboxPage *InboxPage

	ListFilesPartial    *ListFilesPartial
	FileListItemPartial *FileListItemPartial
	FileMetadataPartial *FileMetadataPartial

	ListInboxAssignmentSuggestionsPartial *ListInboxAssignmentSuggestionsPartial
	AssignmentDirectoryListItemPartial    *AssignmentDirectoryListItemPartial
	AssignFileCmd                         *AssignFileCmd

	ShowFilePartial     *FilePartial
	ShowFileTabsPartial *FileTabsPartial
	MoveFileCmd         *MoveFileCmd
	UploadFileCmd       *UploadFileCmd
	MarkAsDoneCmd       *MarkAsDoneCmd
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

		InboxRootPage:          NewInboxRootPage(infra, actions),
		InboxWithSelectionPage: NewInboxWithSelectionPage(infra, actions),

		InboxPage: NewInboxPage(infra, actions),

		ListFilesPartial:                      NewListFilesPartial(infra, actions),
		FileListItemPartial:                   NewFileListItemPartial(infra, actions),
		ListInboxAssignmentSuggestionsPartial: NewListInboxAssignmentSuggestionsPartial(infra, actions),
		AssignmentDirectoryListItemPartial:    NewAssignmentDirectoryListItemPartial(infra, actions),
		AssignFileCmd:                         NewAssignFileCmd(infra, actions),

		ShowFilePartial:     NewFilePartial(infra, actions),
		ShowFileTabsPartial: NewFileTabsPartial(infra, actions),
		MoveFileCmd:         NewMoveFileCmd(infra, actions),
		UploadFileCmd:       NewUploadFileCmd(infra, actions),

		MarkAsDoneCmd: NewMarkAsDoneCmd(infra, actions),
	}

	// uses actions in constructor, thus outside
	actions.FileMetadataPartial = NewFileMetadataPartial(infra, actions)

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.InboxActionsRoute() + path
}
