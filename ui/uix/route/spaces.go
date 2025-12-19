package route

import (
	"fmt"
)

func SpacesRoute() string {
	return "GET /org/{tenant_id}/spaces/"
}

func SpacesActionsRoute() string {
	return "/spaces/"
}

func SpacesRoot(tenantID string) string {
	return fmt.Sprintf("/org/%s/spaces/", tenantID)
}

/*
func SpacesRootWithState(data any) string {
	return RootWithState2(SpacesRoot, data)
}
*/
