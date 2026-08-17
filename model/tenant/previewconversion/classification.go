package previewconversion

import (
	"mime"
	"path/filepath"
	"strings"
)

type Family string

const (
	FamilyHTML     Family = "html"
	FamilyMarkdown Family = "markdown"
	FamilyOffice   Family = "office"
)

const (
	HTMLRoute     = "/forms/chromium/convert/html"
	MarkdownRoute = "/forms/chromium/convert/markdown"
	OfficeRoute   = "/forms/libreoffice/convert"
)

type Classification struct {
	Family         Family
	Route          string
	InputFilename  string
	OutputFilename string
}

func NewClassification(
	family Family,
	route string,
	inputFilename string,
	outputFilename string,
) *Classification {
	return &Classification{
		Family:         family,
		Route:          route,
		InputFilename:  inputFilename,
		OutputFilename: outputFilename,
	}
}

func Classify(mimeType, filename string, isDirectory bool) (*Classification, bool) {
	if isDirectory {
		return nil, false
	}

	normalizedMIME := normalizeMIME(mimeType)
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if extension == "pdf" || normalizedMIME == "application/pdf" {
		return nil, false
	}

	family, inputExtension := familyForExtension(extension)
	fromMIME := false
	if family == "" {
		family, inputExtension = familyForMIME(normalizedMIME)
		fromMIME = true
	}
	if family == "" {
		return nil, false
	}

	inputFilename := filepath.Base(filename)
	switch family {
	case FamilyHTML:
		inputFilename = "index.html"
	case FamilyMarkdown:
		inputFilename = "source.md"
	case FamilyOffice:
		if fromMIME {
			baseName := strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename))
			if baseName == "" || baseName == "." {
				baseName = "source"
			}
			inputFilename = baseName + "." + inputExtension
		}
	}

	return NewClassification(
		family,
		routeForFamily(family),
		inputFilename,
		PreviewFilename(filename),
	), true
}

func PreviewFilename(filename string) string {
	baseName := filepath.Base(filename)
	if baseName == "." || baseName == "" {
		return "preview.pdf"
	}
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if baseName == "" {
		baseName = "preview"
	}
	return baseName + ".pdf"
}

func normalizeMIME(mimeType string) string {
	parsed, _, err := mime.ParseMediaType(strings.TrimSpace(mimeType))
	if err == nil {
		return strings.ToLower(parsed)
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
}

func familyForExtension(extension string) (Family, string) {
	switch extension {
	case "html", "htm", "xhtml":
		return FamilyHTML, "html"
	case "md", "markdown", "mdown", "mkdn":
		return FamilyMarkdown, "md"
	case "doc", "docx", "docm", "dot", "dotx", "dotm", "odt", "ott", "fodt", "rtf",
		"wps", "xls", "xlsx", "xlsm", "xlt", "xltx", "xltm", "ods", "ots", "fods", "csv",
		"ppt", "pptx", "pptm", "pot", "potx", "potm", "odp", "otp", "fodp":
		return FamilyOffice, extension
	default:
		return "", ""
	}
}

func familyForMIME(mimeType string) (Family, string) {
	switch mimeType {
	case "text/html", "application/xhtml+xml":
		return FamilyHTML, "html"
	case "text/markdown", "text/x-markdown":
		return FamilyMarkdown, "md"
	case "application/msword", "application/vnd.ms-word", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-word.document.macroenabled.12", "application/vnd.ms-word.template.macroenabled.12",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.template", "application/vnd.oasis.opendocument.text",
		"application/vnd.oasis.opendocument.text-template", "application/vnd.oasis.opendocument.text-flat-xml",
		"application/rtf", "text/rtf":
		return FamilyOffice, officeExtensionForMIME(mimeType, "docx")
	case "application/vnd.ms-excel", "application/vnd.ms-excel.sheet.macroenabled.12",
		"application/vnd.ms-excel.template.macroenabled.12", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.template", "application/vnd.oasis.opendocument.spreadsheet",
		"application/vnd.oasis.opendocument.spreadsheet-template", "application/vnd.oasis.opendocument.spreadsheet-flat-xml", "text/csv":
		return FamilyOffice, officeExtensionForMIME(mimeType, "xlsx")
	case "application/vnd.ms-powerpoint", "application/vnd.ms-powerpoint.presentation.macroenabled.12",
		"application/vnd.ms-powerpoint.template.macroenabled.12", "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.openxmlformats-officedocument.presentationml.template", "application/vnd.oasis.opendocument.presentation",
		"application/vnd.oasis.opendocument.presentation-template", "application/vnd.oasis.opendocument.presentation-flat-xml":
		return FamilyOffice, officeExtensionForMIME(mimeType, "pptx")
	default:
		return "", ""
	}
}

func officeExtensionForMIME(mimeType, fallback string) string {
	switch mimeType {
	case "application/msword", "application/vnd.ms-word":
		return "doc"
	case "application/vnd.ms-word.document.macroenabled.12":
		return "docm"
	case "application/vnd.ms-word.template.macroenabled.12":
		return "dotm"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.template":
		return "dotx"
	case "application/vnd.oasis.opendocument.text":
		return "odt"
	case "application/vnd.oasis.opendocument.text-template":
		return "ott"
	case "application/vnd.oasis.opendocument.text-flat-xml":
		return "fodt"
	case "application/rtf", "text/rtf":
		return "rtf"
	case "application/vnd.ms-excel":
		return "xls"
	case "application/vnd.ms-excel.sheet.macroenabled.12":
		return "xlsm"
	case "application/vnd.ms-excel.template.macroenabled.12":
		return "xltm"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "xlsx"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.template":
		return "xltx"
	case "application/vnd.oasis.opendocument.spreadsheet":
		return "ods"
	case "application/vnd.oasis.opendocument.spreadsheet-template":
		return "ots"
	case "application/vnd.oasis.opendocument.spreadsheet-flat-xml":
		return "fods"
	case "text/csv":
		return "csv"
	case "application/vnd.ms-powerpoint":
		return "ppt"
	case "application/vnd.ms-powerpoint.presentation.macroenabled.12":
		return "pptm"
	case "application/vnd.ms-powerpoint.template.macroenabled.12":
		return "potm"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "pptx"
	case "application/vnd.openxmlformats-officedocument.presentationml.template":
		return "potx"
	case "application/vnd.oasis.opendocument.presentation":
		return "odp"
	case "application/vnd.oasis.opendocument.presentation-template":
		return "otp"
	case "application/vnd.oasis.opendocument.presentation-flat-xml":
		return "fodp"
	default:
		return fallback
	}
}

func routeForFamily(family Family) string {
	switch family {
	case FamilyHTML:
		return HTMLRoute
	case FamilyMarkdown:
		return MarkdownRoute
	case FamilyOffice:
		return OfficeRoute
	default:
		return ""
	}
}
