package model

import (
	"github.com/simpledms/simpledms/model/tagging"
)

type Attribute struct {
	Tag       *tagging.Tag // TODO can also be tag group
	Protected bool
	Required  bool
	Hidden    bool // only if protected
}
