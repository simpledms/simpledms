package widget

import "testing"

func TestListItemHasSupportingText(t *testing.T) {
	for name, item := range map[string]*ListItem{
		"nil":   {},
		"empty": {SupportingText: Tu("")},
		"text":  {SupportingText: Tu("Supporting text")},
	} {
		t.Run(name, func(t *testing.T) {
			if got := item.HasSupportingText(); got != (name == "text") {
				t.Errorf("HasSupportingText() = %t", got)
			}
		})
	}
}
