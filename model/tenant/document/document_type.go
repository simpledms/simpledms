package document

// TODO global properties

type DocumentType struct {
	Name       string
	Protected  bool
	Properties []*Attribute
}
