package widget

import "html/template"

// DataAttribute is one data-* attribute of DataAttributes. Name is the part after "data-".
type DataAttribute struct {
	Name  template.HTMLAttr
	Value *Text
}
