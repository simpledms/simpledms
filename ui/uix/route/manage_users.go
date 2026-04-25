package route

import (
	"fmt"
)

func ManageUsersOfSpaceRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/manage-users/"
}

func ManageUsersOfSpaceActionsRoute() string {
	return "/manage-space-users/"
}

func ManageUsersOfSpace(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/manage-users/", tenantID, spaceID)
}
