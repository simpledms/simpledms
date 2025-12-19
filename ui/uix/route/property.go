package route

import (
	"fmt"
)

func PropertiesRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/fields/"
}

func PropertyActionsRoute() string {
	return "/fields/"
}

func PropertiesRoot(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/fields", tenantID, spaceID)
}
