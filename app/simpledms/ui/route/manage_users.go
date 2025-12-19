package route

import (
	"fmt"
)

func ManageUsersOfSpaceRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/manage-users/"
}

func ManageUsersOfTenantRoute() string {
	return "GET /org/{tenant_id}/manage-users/"
}

func ManageUsersOfSpaceActionsRoute() string {
	return "/manage-space-users/"
}

func ManageUsersOfTenantActionsRoute() string {
	return "/manage-org-users/"
}

func ManageUsersOfTenant(tenantID string) string {
	return fmt.Sprintf("/org/%s/manage-users/", tenantID)
}

func ManageUsersOfSpace(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/manage-users/", tenantID, spaceID)
}
