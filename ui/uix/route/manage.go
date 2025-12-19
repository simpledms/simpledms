package route

import (
	"fmt"
)

func ManageDocumentTypesRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/document-types/"
}
func ManageDocumentTypesRouteWithSelection() string {
	return "GET /org/{tenant_id}/space/{space_id}/document-types/{id}"
}

func ManageDocumentTypesActionsRoute() string {
	return "/document-types/"
}

func ManageDocumentTypes(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/document-types/", tenantID, spaceID)
}

func ManageDocumentTypesWithSelection(tenantID, spaceID string, id int64) string {
	return fmt.Sprintf("/org/%s/space/%s/document-types/%d", tenantID, spaceID, id)
}

func ManageTagsRoute() string {
	return "GET /org/{tenant_id}/space/{space_id}/tags/"
}
func ManageTagsRouteWithSelection() string {
	return "GET /org/{tenant_id}/space/{space_id}/tags/{tag_id}"
}

func ManageTags(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/tags/", tenantID, spaceID)
}

func ManageTagsWithState(state any) func(string, string) string {
	return RootWithState(ManageTags, state)
}

/*
func ManageTag(tenantID, spaceID string, tagID int64) string {
	return fmt.Sprintf("/org/%s/space/%s/tags/%d", tenantID, spaceID, tagID)
}
*/

func ManageProperties(tenantID, spaceID string) string {
	return fmt.Sprintf("/org/%s/space/%s/fields/", tenantID, spaceID)
}
