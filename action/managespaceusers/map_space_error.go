package managespaceusers

import (
	"errors"
	"net/http"

	"github.com/simpledms/simpledms/core/util/e"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
)

func mapSpaceError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, spacemodel.ErrUserAlreadyAssignedToSpace) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "User is already assigned to this space.")
	}

	if errors.Is(err, spacemodel.ErrCannotUnassignYourselfInSpace) {
		return e.NewHTTPErrorf(http.StatusForbidden, "You cannot unassign yourself from a space.")
	}

	return err
}
