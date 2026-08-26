//go:generate go tool enumer -type=FileSource -sql -ent -json -empty_string -output=file_source.gen.go
package filesource

// FileSource describes how a logical file first entered SimpleDMS.
type FileSource int

const (
	UnknownLegacy FileSource = iota
	WebInterface
	PWAOSOpen
	URLImport
	WebDAV
	SystemExtraction
)
