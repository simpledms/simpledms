package model

import (
	"github.com/simpledms/simpledms/app/simpledms/model/tagging"
)

type Document struct {
	DocumentType *DocumentType
	Tags         []*tagging.Tag
	Versions     []*File
}

func (qq *Document) SelectDocumentType(documentType *DocumentType) {
	// TODO
}

func (qq *Document) DeselectDocumentType(documentType *DocumentType) {
	// TODO
}

func (qq *Document) SelectAttribute(attribute *Attribute) {
	// TODO
}
