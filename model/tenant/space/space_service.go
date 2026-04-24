package space

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/library"
)

func Create(
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

	repos := filemodel.NewEntSpaceFileRepositoryFactory().ForSpaceX(spaceCtx)
	_, err = repos.Write.CreateRootDirectory(spaceCtx, name)
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
