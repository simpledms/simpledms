package file

import "github.com/simpledms/simpledms/ctxx"

type SpaceFileRepositoryFactory interface {
	ForSpace(ctx ctxx.Context) (*SpaceFileRepositories, error)
	ForSpaceX(ctx ctxx.Context) *SpaceFileRepositories
}
