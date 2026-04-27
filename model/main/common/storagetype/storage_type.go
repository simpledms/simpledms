//go:generate go tool enumer -type=StorageType -sql -ent -json -empty_string -output=storage_type.gen.go
package storagetype

type StorageType int

const (
	Unknown StorageType = iota
	Local               // TODO or FileSystem?
	S3
)
