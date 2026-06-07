package space

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
)

type SpaceRepository interface {
	UserByPublicID(ctx ctxx.Context, userPublicID string) (*enttenant.User, error)
	UserAssignmentExists(ctx ctxx.Context, spaceID int64, userID int64) (bool, error)
	CreateUserAssignment(ctx ctxx.Context, spaceID int64, userID int64, role spacerole.SpaceRole) error
	UserAssignmentByID(ctx ctxx.Context, spaceID int64, assignmentID int64) (*enttenant.SpaceUserAssignment, error)
	DeleteUserAssignment(ctx ctxx.Context, assignmentID int64) error
	UnassignedUsers(ctx ctxx.Context, spaceID int64) ([]*enttenant.User, error)
}
