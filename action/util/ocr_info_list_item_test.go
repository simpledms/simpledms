package util

import (
	"testing"

	"github.com/simpledms/simpledms/core/ui/widget"
)

func TestOCRInfoListItemOpensDialogAfterOCRSuccess(t *testing.T) {
	dialogAttrs := widget.HTMXAttrs{
		HxPost:        "/-/browse/ocr-content-dialog",
		LoadInPopover: true,
	}
	item := OCRInfoListItem(true, widget.Tu("today"), dialogAttrs)
	if item.HxPost != dialogAttrs.HxPost || !item.LoadInPopover {
		t.Fatalf("expected OCR dialog link, got %#v", item.HTMXAttrs)
	}
	if icon, ok := item.Trailing.(*widget.Icon); !ok || icon.Name != "visibility" {
		t.Fatalf("expected visibility icon, got %#v", item.Trailing)
	}
}

func TestOCRInfoListItemDoesNotOpenDialogBeforeOCRSuccess(t *testing.T) {
	item := OCRInfoListItem(false, widget.Tu("-"), widget.HTMXAttrs{HxPost: "/dialog"})
	if item.HxPost != "" || item.Trailing != nil {
		t.Fatalf("expected pending OCR item without interaction, got %#v", item)
	}
}
