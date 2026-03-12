package library

import (
	"github.com/simpledms/simpledms/model/tenant/document"
	"github.com/simpledms/simpledms/model/tenant/tagging"
)

type Library struct {
	documentTypes []*document.DocumentType
	tags          []*tagging.Tag
}
