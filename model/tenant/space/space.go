package space

import (
	"net/http"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/util/e"
)

type Space struct {
	Data *enttenant.Space
}

func NewSpace(space *enttenant.Space) *Space {
	return &Space{space}
}

// Enable a document type and tags library
// TODO Enable or Subscribe?
func (qq *Space) EnableLibrary() {
	// TODO
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

	err = ctx.TenantCtx().TTx.File.Update().
		SetName(name).
		Where(
			file.SpaceID(spacex.ID),
			file.IsDirectory(true),
			file.IsRootDir(true),
		).
		Exec(spaceCtx)
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
	userx, err := ctx.SpaceCtx().TTx.User.Query().Where(user.PublicID(entx.NewCIText(userPublicID))).Only(ctx)
	if err != nil {
		return err
	}

	isAlreadyAssigned, err := qq.Data.QueryUserAssignment().
		Where(spaceuserassignment.UserID(userx.ID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if isAlreadyAssigned {
		return e.NewHTTPErrorf(http.StatusBadRequest, "User is already assigned to this space.")
	}

	_, err = ctx.SpaceCtx().TTx.SpaceUserAssignment.Create().
		SetSpaceID(qq.Data.ID).
		SetUserID(userx.ID).
		SetRole(role).
		Save(ctx)

	return err
}

func (qq *Space) UnassignUser(ctx ctxx.Context, userAssignmentID int64, actingUserID int64) error {
	assignment, err := qq.Data.QueryUserAssignment().
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
