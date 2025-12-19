package model

import "github.com/simpledms/simpledms/app/simpledms/model/tagging"

type Library struct {
	documentTypes []*DocumentType
	tags          []*tagging.Tag
}
