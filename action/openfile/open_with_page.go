package openfile

import (
	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
)

type OpenWithPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}
