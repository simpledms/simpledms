//go:generate go tool enumer -type=Language -sql -ent -json -empty_string -output=language.gen.go
package language

import (
	languagex "golang.org/x/text/language"
)

type Language int

const (
	// TODO or Primary or Primitive or Core or Basic or Normal?
	// 		`convert to base tag` sounds right
	Unknown Language = iota
	German
	English
	French
	Italian
)

func (qq Language) Tag() languagex.Tag {
	// TODO find a generic way for matching
	switch qq {
	case German:
		return languagex.German
	case English:
		return languagex.English
	case French:
		return languagex.French
	case Italian:
		return languagex.Italian
	default:
		return languagex.English
	}
}

/*
func (qq TagType) FormValue() string {
	return strconv.Itoa(int(qq))
}
*/
