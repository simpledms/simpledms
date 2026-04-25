package space

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/db/entx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/tenant/common/spacerole"
)

type EntSpaceRepository struct{}

var _ SpaceRepository = (*EntSpaceRepository)(nil)

func NewEntSpaceRepository() *EntSpaceRepository {
	return &EntSpaceRepository{}
}

func (qq *EntSpaceRepository) UserByPublicID(ctx ctxx.Context, userPublicID string) (*enttenant.User, error) {
	return ctx.SpaceCtx().TTx.User.Query().Where(user.PublicID(entx.NewCIText(userPublicID))).Only(ctx)
}

func (qq *EntSpaceRepository) UserAssignmentExists(ctx ctxx.Context, spaceID int64, userID int64) (bool, error) {
	return ctx.SpaceCtx().TTx.SpaceUserAssignment.Query().
		Where(
			spaceuserassignment.SpaceID(spaceID),
			spaceuserassignment.UserID(userID),
		).
		Exist(ctx)
}

func (qq *EntSpaceRepository) CreateUserAssignment(
	ctx ctxx.Context,
	spaceID int64,
	userID int64,
	role spacerole.SpaceRole,
) error {
	_, err := ctx.SpaceCtx().TTx.SpaceUserAssignment.Create().
		SetSpaceID(spaceID).
		SetUserID(userID).
		SetRole(role).
		Save(ctx)

	return err
}

func (qq *EntSpaceRepository) UserAssignmentByID(
	ctx ctxx.Context,
	spaceID int64,
	assignmentID int64,
) (*enttenant.SpaceUserAssignment, error) {
	return ctx.SpaceCtx().TTx.SpaceUserAssignment.Query().
		Where(
			spaceuserassignment.SpaceID(spaceID),
			spaceuserassignment.ID(assignmentID),
		).
		Only(ctx)
}

func (qq *EntSpaceRepository) DeleteUserAssignment(ctx ctxx.Context, assignmentID int64) error {
	return ctx.SpaceCtx().TTx.SpaceUserAssignment.DeleteOneID(assignmentID).Exec(ctx)
}

func (qq *EntSpaceRepository) UnassignedUsers(ctx ctxx.Context, spaceID int64) ([]*enttenant.User, error) {
	return ctx.TenantCtx().TTx.User.Query().
		WithSpaceAssignment().
		Where(user.Not(user.HasSpaceAssignmentWith(spaceuserassignment.SpaceID(spaceID)))).
		All(ctx)
}
