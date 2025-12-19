package openfile

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	"github.com/simpledms/simpledms/app/simpledms/common"
)

type OpenWithPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}
