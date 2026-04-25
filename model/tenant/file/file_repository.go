package file

import "github.com/marcobeierer/go-core/ctxx"

type FileRepository interface {
	GetX(ctx ctxx.Context, id string) *File
	GetWithParentX(ctx ctxx.Context, id string) *FileWithParent
}
