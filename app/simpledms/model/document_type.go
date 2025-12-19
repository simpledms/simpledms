package model

// TODO global properties

type DocumentType struct {
	Name       string
	Protected  bool
	Properties []*Attribute
}
