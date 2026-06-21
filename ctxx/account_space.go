package ctxx

type AccountSpace struct {
	ID       int64
	PublicID string
	Name     string
}

func NewAccountSpace(id int64, publicID string, name string) AccountSpace {
	return AccountSpace{
		ID:       id,
		PublicID: publicID,
		Name:     name,
	}
}
