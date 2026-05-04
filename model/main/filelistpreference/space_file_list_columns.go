package filelistpreference

import "slices"

type SpaceFileListColumns struct {
	PropertyIDs []int64 `json:"property_ids,omitempty"`
	TagGroupIDs []int64 `json:"tag_group_ids,omitempty"`
	ShowTags    bool    `json:"show_tags,omitempty"`
}

func NewSpaceFileListColumns() *SpaceFileListColumns {
	return &SpaceFileListColumns{}
}

func (qq *SpaceFileListColumns) HasPropertyID(propertyID int64) bool {
	return slices.Contains(qq.PropertyIDs, propertyID)
}

func (qq *SpaceFileListColumns) HasTagGroupID(tagGroupID int64) bool {
	return slices.Contains(qq.TagGroupIDs, tagGroupID)
}

func (qq *SpaceFileListColumns) TogglePropertyID(propertyID int64) {
	if propertyID <= 0 {
		return
	}

	for qi, id := range qq.PropertyIDs {
		if id == propertyID {
			qq.PropertyIDs = slices.Delete(qq.PropertyIDs, qi, qi+1)
			return
		}
	}

	qq.PropertyIDs = append(qq.PropertyIDs, propertyID)
	slices.Sort(qq.PropertyIDs)
	qq.PropertyIDs = slices.Compact(qq.PropertyIDs)
}

func (qq *SpaceFileListColumns) ToggleTagGroupID(tagGroupID int64) {
	if tagGroupID <= 0 {
		return
	}

	for qi, id := range qq.TagGroupIDs {
		if id == tagGroupID {
			qq.TagGroupIDs = slices.Delete(qq.TagGroupIDs, qi, qi+1)
			return
		}
	}

	qq.TagGroupIDs = append(qq.TagGroupIDs, tagGroupID)
	slices.Sort(qq.TagGroupIDs)
	qq.TagGroupIDs = slices.Compact(qq.TagGroupIDs)
}
