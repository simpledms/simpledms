package widget

import (
	"fmt"
	"strings"
)

type Badge struct {
	Widget[Badge]
	HTMXAttrs
	Value     int
	IsInline  bool
	IsInverse bool
}

func (qq *Badge) GetValue() string {
	if qq.Value < 1000 {
		return fmt.Sprintf("%d", qq.Value)
	}
	return "999+"
}

func (qq *Badge) GetClass() string {
	classes := []string{}
	// classes := []string{"badge", "primary"}
	if qq.IsInline {
		// classes = append(classes, "none")
	}
	return strings.Join(classes, " ")

}
