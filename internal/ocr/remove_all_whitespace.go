package ocr

import (
	"regexp"
	"strings"
)

var regexpEndsAlphanumeric = regexp.MustCompile("[a-zA-Z0-9]$")

func removeAllWhitespace(text string) string {
	parsedContentSlice := strings.Split(text, "\n")
	var contentSlice []string

	// remove all whitespace
	for _, paragraph := range parsedContentSlice {
		trimmed := strings.TrimSpace(paragraph)

		if trimmed == "" {
			continue
		}

		// add dot if last char is alphanumeric
		// TODO not a perfect solution, sometimes to many dots are added,
		// 		for example if paragraph was not detected correctly by OCR
		if regexpEndsAlphanumeric.MatchString(trimmed) {
			trimmed += "."
		}

		contentSlice = append(contentSlice, trimmed)
	}

	return strings.Join(contentSlice, " ")
}
