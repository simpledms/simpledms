package route

import (
	"fmt"
)

func InboxRoute(withPath bool, post bool) string {
	if withPath {
		return "GET /org/{tenant_id}/space/{space_id}/inbox/{file_id}"
	}
	// moved to /upload/
	// if post {
	// for use with PWA share target, not necessary to have route with
	// file_id
	// return "POST /inbox/"
	// }
	return "GET /org/{tenant_id}/space/{space_id}/inbox/"
}

func InboxActionsRoute() string {
	return "/inbox/"
}

func InboxRoot(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/inbox/", tenantID, spaceID)
}

// don't use this for links, just to set URL in browser
func InboxRootWithState(state any) func(string, string) string {
	return RootWithState(InboxRoot, state)
}

// TODO impl query support
func Inbox(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("/org/%s/space/%s/inbox/%s", tenantID, spaceID, fileID)
}

// don't use this for links, just to set URL in browser
func InboxWithState(data any) func(string, string, string) string {
	return WithState(Inbox, data)
}
