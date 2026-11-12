package widget

import (
	"fmt"
	"html/template"
	"maps"
	"regexp"
	"slices"
)

// dataAttributeNamePattern matches names that dataset exposes in camelCase, such as
// "copied-message" for dataset.copiedMessage.
var dataAttributeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

type DataAttributes struct {
	Widget[DataAttributes]

	Hidden bool
	Values map[string]*Text
}

// Attributes returns the values sorted by name. Templates cannot escape attribute names, so
// each name is validated before it is marked safe.
func (qq *DataAttributes) Attributes() ([]DataAttribute, error) {
	names := slices.Sorted(maps.Keys(qq.Values))
	attributes := make([]DataAttribute, 0, len(names))
	for _, name := range names {
		if !dataAttributeNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid data attribute name %q", name)
		}
		attributes = append(attributes, DataAttribute{
			Name:  template.HTMLAttr(name),
			Value: qq.Values[name],
		})
	}
	return attributes, nil
}
