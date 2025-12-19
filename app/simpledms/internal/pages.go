package internal

import (
	"github.com/simpledms/simpledms/app/simpledms/action"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/page"
)

// Deprecated: implement directly as action
type Pages struct {
	Browse              *page.Browse
	BrowseWithSelection *page.BrowseWithSelection
	// Find                *page.Find
	// FindWithSelection *page.FindWithSelection
	// TODO does this belong here?
	Download           *page.Download
	Inbox              *page.Inbox
	InboxWithSelection *page.InboxWithSelection

	ManageDocumentTypes *page.ManageDocumentTypes
	// ManageTags          *page.ManageTags
}

// Deprecated: implement directly as action
func NewPages(
	infra *common.Infra,
	actions *action.Actions,
) *Pages {
	// TODO refactor
	return &Pages{
		Browse:              page.NewBrowse(infra, actions.Browse),
		BrowseWithSelection: page.NewBrowseWithSelection(infra, actions.Browse),
		// Find:                page.NewFind(infra, actions.Find),
		// FindWithSelection:  page.NewFindWithSelection(infra, actions.Find),
		Download:           page.NewDownload(infra, actions),
		Inbox:              page.NewInbox(infra, actions.Inbox),
		InboxWithSelection: page.NewInboxWithSelection(infra, actions.Inbox),

		ManageDocumentTypes: page.NewManageDocumentTypes(infra, actions.DocumentType),
		// ManageTags:          page.NewManageTags(infra, actions.Tagging),
	}
}
