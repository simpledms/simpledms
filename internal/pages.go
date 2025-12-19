package internal

import (
	"github.com/simpledms/simpledms/action"
	"github.com/simpledms/simpledms/common"
	page2 "github.com/simpledms/simpledms/ui/uix/page"
)

// Deprecated: implement directly as action
type Pages struct {
	Browse              *page2.Browse
	BrowseWithSelection *page2.BrowseWithSelection
	// Find                *page.Find
	// FindWithSelection *page.FindWithSelection
	// TODO does this belong here?
	Download           *page2.Download
	Inbox              *page2.Inbox
	InboxWithSelection *page2.InboxWithSelection

	ManageDocumentTypes *page2.ManageDocumentTypes
	// ManageTags          *page.ManageTags
}

// Deprecated: implement directly as action
func NewPages(
	infra *common.Infra,
	actions *action.Actions,
) *Pages {
	// TODO refactor
	return &Pages{
		Browse:              page2.NewBrowse(infra, actions.Browse),
		BrowseWithSelection: page2.NewBrowseWithSelection(infra, actions.Browse),
		// Find:                page.NewFind(infra, actions.Find),
		// FindWithSelection:  page.NewFindWithSelection(infra, actions.Find),
		Download:           page2.NewDownload(infra, actions),
		Inbox:              page2.NewInbox(infra, actions.Inbox),
		InboxWithSelection: page2.NewInboxWithSelection(infra, actions.Inbox),

		ManageDocumentTypes: page2.NewManageDocumentTypes(infra, actions.DocumentType),
		// ManageTags:          page.NewManageTags(infra, actions.Tagging),
	}
}
