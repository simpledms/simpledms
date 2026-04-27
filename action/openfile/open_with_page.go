package openfile

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
)

type OpenWithPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}
