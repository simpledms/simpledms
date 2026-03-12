package model

import "github.com/simpledms/simpledms/model/tenant/tagging"

type Library struct {
	documentTypes []*DocumentType
	tags          []*tagging.Tag
}
