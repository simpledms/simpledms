//go:generate go tool enumer -type=Country -sql -ent -json -empty_string -output=country.gen.go
package country

type Country int

const (
	// TODO or Primary or Primitive or Core or Basic or Normal?
	// 		`convert to base tag` sounds right
	Unknown Country = iota
	Austria
	Germany
	Switzerland
)

/*
func (qq TagType) FormValue() string {
	return strconv.Itoa(int(qq))
}
*/
