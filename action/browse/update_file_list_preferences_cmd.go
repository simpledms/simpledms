package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/main/filelistpreference"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type UpdateFileListPreferencesCmdData struct {
	ViewMode      string `json:"ViewMode,omitempty"`
	BuiltInColumn string `json:"BuiltInColumn,omitempty"`
	ShowTags      *bool  `json:"ShowTags,omitempty"`
	PropertyID    int64  `json:"PropertyID,omitempty"`
	TagGroupID    int64  `json:"TagGroupID,omitempty"`
}

type UpdateFileListPreferencesCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUpdateFileListPreferencesCmd(
	infra *common.Infra,
	actions *Actions,
) *UpdateFileListPreferencesCmd {
	return &UpdateFileListPreferencesCmd{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("update-file-list-preferences-cmd"),
			false,
		),
	}
}

func (qq *UpdateFileListPreferencesCmd) Data() *UpdateFileListPreferencesCmdData {
	return &UpdateFileListPreferencesCmdData{}
}

func (qq *UpdateFileListPreferencesCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[UpdateFileListPreferencesCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	preferences := filelistpreference.NewFileListPreferencesFromValue(ctx.MainCtx().Account.FileListPreferences)
	if data.ViewMode != "" {
		preferences.SetViewMode(filelistpreference.FileListViewModeString(data.ViewMode))
	}
	if data.BuiltInColumn != "" {
		column, found := filelistpreference.FileListColumnString(data.BuiltInColumn)
		if found {
			preferences.ToggleBuiltInColumn(column)
		}
	}
	if data.ShowTags != nil {
		preferences.SetSpaceTags(ctx.SpaceCtx().SpaceID, *data.ShowTags)
	}
	if data.PropertyID > 0 && qq.canUseProperty(ctx, data.PropertyID) {
		preferences.ToggleSpacePropertyID(ctx.SpaceCtx().SpaceID, data.PropertyID)
	}
	if data.TagGroupID > 0 && qq.canUseTagGroup(ctx, data.TagGroupID) {
		preferences.ToggleSpaceTagGroupID(ctx.SpaceCtx().SpaceID, data.TagGroupID)
	}

	accountx, err := ctx.MainCtx().MainTx.Account.UpdateOneID(ctx.MainCtx().Account.ID).
		SetFileListPreferences(*preferences).
		Save(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	ctx.MainCtx().Account = accountx

	return nil
}

func (qq *UpdateFileListPreferencesCmd) canUseProperty(ctx ctxx.Context, propertyID int64) bool {
	return ctx.SpaceCtx().TTx.Property.Query().Where(property.ID(propertyID)).ExistX(ctx)
}

func (qq *UpdateFileListPreferencesCmd) canUseTagGroup(ctx ctxx.Context, tagGroupID int64) bool {
	return ctx.SpaceCtx().TTx.Tag.Query().Where(tag.ID(tagGroupID), tag.TypeEQ(tagtype.Group)).ExistX(ctx)
}
