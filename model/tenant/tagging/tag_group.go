package tagging

type TagGroup struct {
	Tags          []*Tag
	Multiselect   bool // TODO inverse? SingleSelect?
	AttributeOnly bool // TODO name? No Standalone
}

func (qq *TagGroup) AddTag(tag *Tag) {
	// TODO in db
	qq.Tags = append(qq.Tags, tag)
}
