package route

import (
	"fmt"
)

func ManageUsersOfTenantRoute() string {
	return "GET /org/{tenant_id}/manage-users/"
}

func ManageUsersOfTenantActionsRoute() string {
	return "/manage-org-users/"
}

func ManageUsersOfTenant(tenantID string) string {
	return fmt.Sprintf("/org/%s/manage-users/", tenantID)
}
