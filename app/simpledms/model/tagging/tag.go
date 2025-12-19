package tagging

import (
	"fmt"

	"github.com/simpledms/simpledms/app/simpledms/enttenant"
)

type Tag struct {
	tag *enttenant.Tag
}

func NewTag(tag *enttenant.Tag) *Tag {
	return &Tag{tag: tag}
}

// TODO indicate if composed?
func (qq *Tag) String() string {
	if qq.tag.Edges.Group != nil {
		return fmt.Sprintf("%s: %s", qq.tag.Edges.Group.Name, qq.tag.Name)
	}
	return qq.tag.Name
}
