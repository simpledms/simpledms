package space

import (
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

type Space struct {
	Data       *enttenant.Space
	repository SpaceRepository
}

func NewSpace(space *enttenant.Space) *Space {
	return NewSpaceWithRepository(space, NewEntSpaceRepository())
}

func NewSpaceWithRepository(space *enttenant.Space, repository SpaceRepository) *Space {
	return &Space{
		Data:       space,
		repository: repository,
	}
}

func (qq *Space) Edit(ctx ctxx.Context, name string, description string) error {
	spacex, err := qq.Data.Update().
		SetName(name).
		SetDescription(description).
		Save(ctx)
	if err != nil {
		return err
	}
	qq.Data = spacex

	spaceCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), spacex)
	rootDirID := spaceCtx.SpaceRootDir().ID
	repos := filemodel.NewEntSpaceFileRepositoryFactory().ForSpaceX(spaceCtx)

	err = repos.Write.RenameFileByIDX(spaceCtx, rootDirID, name)
	if err != nil {
		return err
	}

	return nil
}

func (qq *Space) Delete(ctx ctxx.Context, deleter *enttenant.User) error {
	update := qq.Data.Update().
		SetDeletedAt(time.Now())

	if deleter != nil {
		update.SetDeleter(deleter)
	}

	spacex, err := update.Save(ctx)
	if err != nil {
		return err
	}
	qq.Data = spacex

	return nil
}

func (qq *Space) AssignUser(ctx ctxx.Context, userPublicID string, role spacerole.SpaceRole) error {
	userx, err := qq.repository.UserByPublicID(ctx, userPublicID)
	if err != nil {
		return err
	}

	isAlreadyAssigned, err := qq.repository.UserAssignmentExists(ctx, qq.Data.ID, userx.ID)
	if err != nil {
		return err
	}
	if isAlreadyAssigned {
		return ErrUserAlreadyAssignedToSpace
	}

	return qq.repository.CreateUserAssignment(ctx, qq.Data.ID, userx.ID, role)
}

func (qq *Space) UnassignUser(ctx ctxx.Context, userAssignmentID int64, actingUserID int64) error {
	assignment, err := qq.repository.UserAssignmentByID(ctx, qq.Data.ID, userAssignmentID)
	if err != nil {
		return err
	}

	if assignment.UserID == actingUserID {
		return ErrCannotUnassignYourselfInSpace
	}

	return qq.repository.DeleteUserAssignment(ctx, userAssignmentID)
}
