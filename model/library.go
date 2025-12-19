package model

import "github.com/simpledms/simpledms/model/tagging"

type Library struct {
	documentTypes []*DocumentType
	tags          []*tagging.Tag
}
