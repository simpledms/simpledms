package route

import (
	"fmt"
)

func BrowseRoute(withPath bool) string {
	if withPath {
		return "GET /org/{tenant_id}/space/{space_id}/browse/{dir_id}"
	}
	return "GET /org/{tenant_id}/space/{space_id}/browse/"
}

func BrowseRouteWithSelection() string {
	// if withTab {
	// return "GET /browse/{dir_id}/file/{file_id}/tab/{tab}"
	// }
	return "GET /org/{tenant_id}/space/{space_id}/browse/{dir_id}/file/{file_id}"
}

func BrowseActionsRoute() string {
	return "/browse/"
}

func BrowseRoot(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/browse/", tenantID, spaceID)
}

// TODO refactor to return url?
func Browse(tenantID, spaceID, dirID string) string {
	return fmt.Sprintf("/org/%s/space/%s/browse/%s", tenantID, spaceID, dirID)
}

func BrowseFile(tenantID, spaceID, dirID, fileID string) string {
	route := fmt.Sprintf("/org/%s/space/%s/browse/%s/file/%s", tenantID, spaceID, dirID, fileID)
	// if tab != "" {
	// route += fmt.Sprintf("?tab=%s", tab)
	// }
	return route
}

// don't use this for links, just to set URL in browser
func BrowseFileWithState(data any) func(string, string, string, string) string {
	return WithState2(BrowseFile, data)
}

// don't use this for links, just to set URL in browser
func BrowseWithState(data any) func(string, string, string) string {
	// FIXME safety?
	return WithState(Browse, data)
}

func BrowseRootWithState(data any) func(string, string) string {
	return RootWithState(BrowseRoot, data)
}

/*
// Deprecated: use Browse with new query support instead
func BrowseWithActiveTab(activeTab string) func(int64) string {
	return func(fileID int64) string {
		if activeTab == "" {
			return Browse(fileID)
		}

		// TODO use query for tab...
		return fmt.Sprintf("%s?tab=%s", Browse(fileID), activeTab)
	}
}
*/
