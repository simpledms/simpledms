//go:generate go tool enumer -type=FieldType -sql -ent -json -empty_string -output=field_type.gen.go
package fieldtype

type FieldType int

const (
	Unknown FieldType = iota
	Text
	Number
	Money
	// DecimalNumber // float // probably shouldn't be exposed to user because of the issues/edgecases it comes with
	Date
	Checkbox // instead of boolean? cannot handle nil value, but necessary?

	// can help keep groups in check, but needs also nil value (not set);
	// paid or not should ideally be a Date type (paid on)
	// Boolean // TODO good idea or not? could also be solved with 2 tags in a group
	// Textarea // TODO or MultiLineText necessary in addition to Text? or is it the same? for example for Notes
	// DateTime
	// Time
	// Enum: Tag groups are kind of enums
)
