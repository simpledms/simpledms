package document

import (
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/tagging"
)

type Document struct {
	DocumentType *DocumentType
	Tags         []*tagging.Tag
	Versions     []*filemodel.File
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
