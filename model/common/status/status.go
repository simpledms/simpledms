//go:generate go tool enumer -type=Status -sql -ent -json -empty_string -output=status.gen.go
package status

type Status int

const (
	Unknown Status = iota
	Pending
	Accepted
	Rejected
)
