package trash

import (
	"github.com/simpledms/simpledms/action/browse"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	Browse *browse.Actions

	TrashRootPage          *TrashRootPage
	TrashWithSelectionPage *TrashWithSelectionPage

	TrashListPartial            *TrashListPartial
	TrashContextMenuPartial     *TrashContextMenuPartial
	FileDetailsSideSheetPartial *FileDetailsSideSheetPartial
	FileTabsPartial             *FileTabsPartial
	FileMetadataPartial         *FileMetadataPartial
	FileTagsPartial             *FileTagsPartial
	FileInfoPartial             *FileInfoPartial

	RestoreFileCmd *RestoreFileCmd
}

func NewActions(infra *common.Infra, browseActions *browse.Actions) *Actions {
	actions := new(Actions)

	*actions = Actions{
		Browse: browseActions,

		TrashRootPage:          NewTrashRootPage(infra, actions),
		TrashWithSelectionPage: NewTrashWithSelectionPage(infra, actions),

		TrashListPartial:            NewTrashListPartial(infra, actions),
		TrashContextMenuPartial:     NewTrashContextMenuPartial(actions),
		FileDetailsSideSheetPartial: NewFileDetailsSideSheetPartial(infra, actions),
		FileTabsPartial:             NewFileTabsPartial(infra, actions),
		FileMetadataPartial:         NewFileMetadataPartial(infra, actions),
		FileTagsPartial:             NewFileTagsPartial(infra, actions),
		FileInfoPartial:             NewFileInfoPartial(infra, actions),

		RestoreFileCmd: NewRestoreFileCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.TrashActionsRoute() + path
}
