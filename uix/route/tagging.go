package route

import (
	"fmt"
)

func TaggingActionsRoute() string {
	return "/tagging/"
}

func Tagging(tenantID string) string {
	return fmt.Sprintf("/org/%s/tagging/", tenantID)
}
