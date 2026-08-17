//go:generate go tool enumer -type=Status -sql -ent -json -empty_string -output=status.gen.go
package previewconversion

type Status int

const (
	Pending Status = iota + 1
	Processing
	Ready
	Failed
)
