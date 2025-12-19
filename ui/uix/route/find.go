package route

import (
	"fmt"
)

/*
func FindRoute(withPath bool) string {
	if withPath {
		return "GET /find/{file_id}"
	}
	return "GET /find/"
}
*/

func FindActionsRoute() string {
	return "/find/"
}

func FindRoot(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/find/", tenantID, spaceID)
}

/*
func Find(id string) string {
	if strings.HasPrefix(id, "{") {
		return "GET /find/" + id
	}
	// TODO byte conversion safe?
	return "/find/" + base64.URLEncoding.EncodeToString([]byte(id))
}
*/

func Find(tenantID, spaceID, fileID string) string {
	return fmt.Sprintf("%s%s", FindRoot(tenantID, spaceID), fileID)
}

func FindRootWithState(data any) func(string, string) string {
	// FIXME safety?
	return RootWithState(FindRoot, data)
}

func FindWithState(data any) func(string, string, string) string {
	return WithState(Find, data)
}
