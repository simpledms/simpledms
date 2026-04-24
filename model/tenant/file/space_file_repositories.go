package file

type SpaceFileRepositories struct {
	Read  FileReadRepository
	Write FileWriteRepository
	Query FileQueryRepository
}

func NewSpaceFileRepositories(
	read FileReadRepository,
	write FileWriteRepository,
	query FileQueryRepository,
) *SpaceFileRepositories {
	return &SpaceFileRepositories{
		Read:  read,
		Write: write,
		Query: query,
	}
}
