package route

import (
	"fmt"
)

func SelectSpaceRoute(withUploadToken bool) string {
	// if post {
	// for use with PWA share target, not necessary to have route with
	// file_id
	// return "POST /open-file/"
	// }
	if withUploadToken {
		return "GET /open-file/select-space/{upload_token}"
	}
	return "GET /open-file/select-space/"
}

func SelectSpace(tokenURL string) string {
	return fmt.Sprintf("/open-file/select-space/%s", tokenURL)
}

func OpenFileActionsRoute() string {
	return "/open-file/"
}
