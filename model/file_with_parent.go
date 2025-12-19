package model

import (
	"github.com/simpledms/simpledms/ctxx"
)

type FileWithParent struct {
	*File
}

func NewFileWithParent(file *File) *FileWithParent {
	return &FileWithParent{
		File: file,
	}
}

func (qq *FileWithParent) Parent(ctx ctxx.Context) *File {
	if qq.Data.ParentID == 0 {
		panic("file has no parent")
	}

	// TODO is this okay or use a cache in ctx?
	if qq.nilableParent != nil {
		return qq.nilableParent
	}

	if qq.Data.Edges.Parent != nil {
		qq.nilableParent = NewFile(qq.Data.Edges.Parent)
		return qq.nilableParent
	}

	// TODO does this set Edges?
	parentx := qq.Data.QueryParent().OnlyX(ctx)
	qq.nilableParent = NewFile(parentx)

	return qq.nilableParent
}
