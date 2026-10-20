package file

import "github.com/simpledms/simpledms/ctxx"

type FileRepository interface {
	Get(ctx ctxx.Context, id string) (*File, error)
	GetWithParent(ctx ctxx.Context, id string) (*FileWithParent, error)
}
