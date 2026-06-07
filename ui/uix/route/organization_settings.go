package route

import "fmt"

func OrganizationSettingsRoute() string {
	return "GET /org/{tenant_id}/settings/"
}

func OrganizationSettings(tenantID string) string {
	return fmt.Sprintf("/org/%s/settings/", tenantID)
}
