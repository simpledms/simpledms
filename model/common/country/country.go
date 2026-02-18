//go:generate go tool enumer -type=Country -sql -ent -json -empty_string -output=country.gen.go
package country

type Country int

const (
	// TODO or Primary or Primitive or Core or Basic or Normal?
	// 		`convert to base tag` sounds right
	Unknown Country = iota
	Austria
	Belgium
	Bulgaria
	Croatia
	Cyprus
	CzechRepublic
	Denmark
	Estonia
	Finland
	France
	Germany
	Greece
	Hungary
	Iceland
	Ireland
	Italy
	Latvia
	Liechtenstein
	Lithuania
	Luxembourg
	Malta
	Netherlands
	Norway
	Other
	Poland
	Portugal
	Romania
	Slovakia
	Slovenia
	Spain
	Sweden
	Switzerland
)

/*
func (qq TagType) FormValue() string {
	return strconv.Itoa(int(qq))
}
*/
