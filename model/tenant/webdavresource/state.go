//go:generate go tool enumer -type=State -sql -ent -json -empty_string -output=state.gen.go
package webdavresource

// State describes the lifecycle of a WebDAV ingestion alias.
type State int

const (
	Uploading State = iota
	Active
	CleanupPending
)
