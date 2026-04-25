package tagging

import (
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

type TagRepository interface {
	AssignTagToFile(ctx ctxx.Context, fileID int64, tagID int64, spaceID int64) error
	UnassignTagFromFile(ctx ctxx.Context, fileID int64, tagID int64) error
	UnassignTagFromFileInSpace(ctx ctxx.Context, fileID int64, tagID int64, spaceID int64) error
	SubTagWithChildren(ctx ctxx.Context, subTagID int64) (*enttenant.Tag, error)
	CreateTag(ctx ctxx.Context, spaceID int64, groupTagID int64, name string, typex tagtype.TagType) (*enttenant.Tag, error)
	UpdateTagName(ctx ctxx.Context, tagID int64, name string) (*enttenant.Tag, error)
	DeleteTag(ctx ctxx.Context, tagID int64) error
	TagByID(ctx ctxx.Context, tagID int64) (*enttenant.Tag, error)
	FileByID(ctx ctxx.Context, fileID int64) (*enttenant.File, error)
	FileHasTagAssignment(ctx ctxx.Context, fileID int64, tagID int64) (bool, error)
	ClearTagGroup(ctx ctxx.Context, tagID int64) error
	SetTagGroup(ctx ctxx.Context, tagID int64, groupTagID int64) error
	AddSubTag(ctx ctxx.Context, superTagID int64, subTagID int64) (*enttenant.Tag, error)
	RemoveSubTag(ctx ctxx.Context, superTagID int64, subTagID int64) (*enttenant.Tag, error)
	GroupTags(ctx ctxx.Context) ([]*enttenant.Tag, error)
}
