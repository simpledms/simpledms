package util

import "github.com/simpledms/simpledms/core/ui/widget"

// OCRInfoListItem returns a dialog link after OCR succeeds.
func OCRInfoListItem(
	hasOCRSuccess bool,
	ocrSucceededAt *widget.Text,
	dialogAttrs widget.HTMXAttrs,
) *widget.ListItem {
	item := &widget.ListItem{
		Headline:       widget.T("OCR succeeded at"),
		SupportingText: ocrSucceededAt,
	}
	if !hasOCRSuccess {
		return item
	}

	item.HTMXAttrs = dialogAttrs
	item.Trailing = widget.NewIcon("visibility")
	return item
}
