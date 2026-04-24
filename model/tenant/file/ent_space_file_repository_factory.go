package file

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/e"
)

type EntSpaceFileRepositoryFactory struct{}

var _ SpaceFileRepositoryFactory = (*EntSpaceFileRepositoryFactory)(nil)

func NewEntSpaceFileRepositoryFactory() *EntSpaceFileRepositoryFactory {
	return &EntSpaceFileRepositoryFactory{}
}

func (qq *EntSpaceFileRepositoryFactory) ForSpace(ctx ctxx.Context) (*SpaceFileRepositories, error) {
	if !ctx.IsSpaceCtx() || ctx.SpaceCtx() == nil || ctx.SpaceCtx().Space == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Space context is required.")
	}

	spaceID := ctx.SpaceCtx().Space.ID

	return NewSpaceFileRepositories(
		NewEntSpaceFileReadRepository(spaceID),
		NewEntSpaceFileWriteRepository(spaceID),
		NewEntSpaceFileQueryRepository(spaceID),
	), nil
}

func (qq *EntSpaceFileRepositoryFactory) ForSpaceX(ctx ctxx.Context) *SpaceFileRepositories {
	repos, err := qq.ForSpace(ctx)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	return repos
}
