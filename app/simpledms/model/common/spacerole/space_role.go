//go:generate go tool enumer -type=SpaceRole -sql -ent -json -empty_string -output=space_role.gen.go
package spacerole

type SpaceRole int

const (
	User SpaceRole = iota + 1 // was Editor
	Owner
	// Viewer SpaceRole = iota + 1
)
