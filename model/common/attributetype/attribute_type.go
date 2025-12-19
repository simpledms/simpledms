//go:generate go tool enumer -type=AttributeType -sql -ent -json -empty_string -output=attribute_type.gen.go
package attributetype

type AttributeType int

const (
	Tag AttributeType = iota + 1
	Field
)
