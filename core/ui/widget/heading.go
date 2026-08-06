package widget

type HeadingType string

const (
	// Label
	// Body
	HeadingTypeTitleSm    = "title-sm"
	HeadingTypeTitleMd    = "title-md"
	HeadingTypeTitleLg    = "title-lg"
	HeadingTypeHeadlineSm = "headline-sm"
	HeadingTypeHeadlineMd = "headline-md"
	HeadingTypeHeadlineLg = "headline-lg"
	// HeadingTypeDisplaySm  = "display-sm"
	// HeadingTypeDisplayMd  = "display-md"
	// HeadingTypeDisplayLg  = "display-lg"
)

// TODO split up into Title, Headline and Display?
//
// https://m3.material.io/styles/typography/applying-type
type Heading struct {
	Widget[Heading]

	Type HeadingType
	Text *Text
}

func H(typex HeadingType, text *Text) *Heading {
	return &Heading{
		Text: text,
		Type: typex,
	}
}

func Hf(typex HeadingType, format string, a ...any) *Heading {
	return &Heading{
		Text: Tf(format, a...),
		Type: typex,
	}
}
