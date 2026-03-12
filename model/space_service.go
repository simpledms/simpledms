package model

import (
	"net/http"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/model/library"
	"github.com/simpledms/simpledms/util/e"
)

type SpaceService struct{}

func NewSpaceService() *SpaceService {
	return &SpaceService{}
}

func (qq *SpaceService) Create(
	ctx ctxx.Context,
	name string,
	description string,
	addMeAsSpaceOwner bool,
	libraryTemplateKeys []string,
) (*enttenant.Space, error) {
	isDefault := false

	spacex, err := ctx.TenantCtx().TTx.Space.Create().
		SetName(name).
		SetDescription(description).
		SetIsFolderMode(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	spaceCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), spacex)

	if addMeAsSpaceOwner {
		_, err = ctx.TenantCtx().TTx.SpaceUserAssignment.
			Create().
			SetSpaceID(spacex.ID).
			SetUserID(ctx.TenantCtx().User.ID).
			SetRole(spacerole.Owner).
			SetIsDefault(isDefault).
			Save(spaceCtx)
		if err != nil {
			return nil, err
		}
	}

	_, err = ctx.TenantCtx().TTx.File.Create().
		SetName(name).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()).
		SetModifiedAt(time.Now()).
		SetSpaceID(spacex.ID).
		SetIsRootDir(true).
		Save(spaceCtx)
	if err != nil {
		return nil, err
	}

	if len(libraryTemplateKeys) > 0 {
		service := library.NewService()
		err = service.ImportBuiltinDocumentTypes(spaceCtx, libraryTemplateKeys, false)
		if err != nil {
			return nil, err
		}
	}

	return spacex, nil
}

func (qq *SpaceService) Edit(
	ctx ctxx.Context,
	spacePublicID string,
	name string,
	description string,
) (*enttenant.Space, error) {
	err := ctx.TenantCtx().TTx.Space.Update().
		SetName(name).
		SetDescription(description).
		Where(space.PublicID(entx.NewCIText(spacePublicID))).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	spacex, err := ctx.TenantCtx().TTx.Space.Query().
		Where(space.PublicID(entx.NewCIText(spacePublicID))).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	spaceCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), spacex)

	err = ctx.TenantCtx().TTx.File.Update().
		SetName(name).
		Where(
			file.SpaceID(spacex.ID),
			file.IsDirectory(true),
			file.IsRootDir(true),
		).
		Exec(spaceCtx)
	if err != nil {
		return nil, err
	}

	return spacex, nil
}

func (qq *SpaceService) Delete(
	ctx ctxx.Context,
	spacePublicID string,
	deleter *enttenant.User,
) error {
	update := ctx.TenantCtx().TTx.Space.Update().
		SetDeletedAt(time.Now()).
		Where(space.PublicID(entx.NewCIText(spacePublicID)))

	if deleter != nil {
		update.SetDeleter(deleter)
	}

	return update.Exec(ctx)
}

func (qq *SpaceService) AssignUser(
	ctx ctxx.Context,
	space *enttenant.Space,
	userPublicID string,
	role spacerole.SpaceRole,
) error {
	userx, err := ctx.SpaceCtx().TTx.User.Query().Where(user.PublicID(entx.NewCIText(userPublicID))).Only(ctx)
	if err != nil {
		return err
	}

	isAlreadyAssigned, err := space.QueryUserAssignment().
		Where(spaceuserassignment.UserID(userx.ID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if isAlreadyAssigned {
		return e.NewHTTPErrorf(http.StatusBadRequest, "User is already assigned to this space.")
	}

	_, err = ctx.SpaceCtx().TTx.SpaceUserAssignment.Create().
		SetSpace(space).
		SetUserID(userx.ID).
		SetRole(role).
		Save(ctx)

	return err
}

func (qq *SpaceService) UnassignUser(
	ctx ctxx.Context,
	space *enttenant.Space,
	userAssignmentID int64,
	actingUserID int64,
) error {
	assignment, err := space.QueryUserAssignment().
		Where(spaceuserassignment.ID(userAssignmentID)).
		Only(ctx)
	if err != nil {
		return err
	}

	if assignment.UserID == actingUserID {
		return e.NewHTTPErrorf(http.StatusForbidden, "You cannot unassign yourself from a space.")
	}

	return ctx.SpaceCtx().TTx.SpaceUserAssignment.DeleteOneID(userAssignmentID).Exec(ctx)
}
