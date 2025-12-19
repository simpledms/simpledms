//go:generate go tool enumer -type=TagType -sql -json -ent -empty_string -output=tag_type.gen.go
package tagtype

type TagType int

const (
	// TODO or Primary or Primitive or Core or Basic or Normal or Standard?
	// 		`convert to base tag` sounds right
	Simple TagType = iota + 1
	Super          // TODO or Power?
	Group
)

/*
func (qq TagType) FormValue() string {
	return strconv.Itoa(int(qq))
}
*/
