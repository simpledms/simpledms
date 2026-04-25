package openfile

import (
	acommon "github.com/simpledms/simpledms/core/action/common"
	"github.com/simpledms/simpledms/core/common"
)

type OpenWithPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}
