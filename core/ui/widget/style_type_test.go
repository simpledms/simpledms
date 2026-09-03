package widget

import "testing"

func TestStyleTypes(t *testing.T) {
	paragraphTests := map[ParagraphStyleType]string{
		ParagraphStyleTypeDefault:         "body-md text-on-surface",
		ParagraphStyleTypeSupporting:      "body-medium text-on-surface-variant",
		ParagraphStyleTypeSupportingSmall: "body-small text-on-surface-variant",
		ParagraphStyleTypeError:           "body-small text-error",
		ParagraphStyleTypeErrorReserved:   "body-small text-error min-h-5",
		ParagraphStyleTypeStatus:          "body-lg text-on-surface p-4",
	}
	for styleType, want := range paragraphTests {
		if got := (&Paragraph{StyleType: styleType}).GetClass(); got != want {
			t.Errorf("Paragraph.GetClass() = %q, want %q", got, want)
		}
	}

	textAreaTests := map[TextAreaStyleType]string{
		TextAreaStyleTypeDefault: "w-full min-h-[12rem] rounded-md border " +
			"border-outline-variant bg-surface-container-low px-4 py-3 title-small font-mono",
		TextAreaStyleTypeCompact: "w-full rounded-md border border-outline-variant " +
			"bg-surface-container-low px-4 py-3 title-small font-mono",
		TextAreaStyleTypeFullHeight: "w-full min-h-full [field-sizing:content] resize-none " +
			"rounded-md border border-outline-variant bg-surface-container-low px-4 py-3 " +
			"title-small font-mono",
	}
	for styleType, want := range textAreaTests {
		if got := (&TextArea{StyleType: styleType}).GetClass(); got != want {
			t.Errorf("TextArea.GetClass() = %q, want %q", got, want)
		}
	}
}
