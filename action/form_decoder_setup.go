package action

import (
	"sync"

	autil "github.com/marcobeierer/go-core/action/util"
	spacerole2 "github.com/simpledms/simpledms/model/tenant/common/spacerole"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

var registerFormDecoderSetupOnce sync.Once

func registerFormDecoderSetup() {
	registerFormDecoderSetupOnce.Do(func() {
		autil.RegisterFormDecoderConverter(tagtype.Simple, tagtype.TagTypeString)
		autil.RegisterFormDecoderConverter(spacerole2.User, spacerole2.SpaceRoleString)
	})
}
