package model

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/iancoleman/strcase"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/attribute"
	"github.com/simpledms/simpledms/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/enttenant/storedfile"
	"github.com/simpledms/simpledms/enttenant/tag"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	mproperty "github.com/simpledms/simpledms/model/property"
	"github.com/simpledms/simpledms/util/e"
)

type File struct {
	// Data *dm.Files

	// how to handle relations? should also return model.X and not ent.X
	Data *enttenant.File // TODO embed or not? make private?

	// Location string
	// DocumentType *DocumentType
	// Tags         []*tagging.Tag
	// Versions     []*File
	nilableCurrentVersion *StoredFile
	nilableParent         *File
}

func NewFile(data *enttenant.File) *File {
	return &File{
		Data: data,
		// data: data,
	}
}

func (qq *File) Filename(ctx ctxx.Context) string {
	if ctx.IsSpaceCtx() && ctx.SpaceCtx().Space.IsFolderMode {
		// return qq.Data.Name
	}

	filenameElems, err := qq.IdentifyingElements(ctx)
	if err != nil {
		return qq.Data.Name
	}

	for i, elem := range filenameElems {
		// TODO safe enough?
		filenameElems[i] = strcase.ToSnake(elem)
	}

	extensionWithDot := filepath.Ext(qq.Data.Name)
	return strings.Join(filenameElems, "-") + extensionWithDot
}

// default is without document type because it is shown as supporting text
func (qq *File) FilenameInApp(ctx ctxx.Context, withDocumentType bool) string {
	if ctx.IsSpaceCtx() && ctx.SpaceCtx().Space.IsFolderMode {
		// return qq.Data.Name
	}

	filenameElems, err := qq.IdentifyingElements(ctx)
	if err != nil {
		return qq.Data.Name
	}

	if withDocumentType {
		return strings.Join(filenameElems, ", ")
	}

	if len(filenameElems) <= 1 {
		return qq.Data.Name
	}

	// remove leading document type
	return strings.Join(filenameElems[1:], ", ")
}

func (qq *File) IdentifyingElements(ctx ctxx.Context) ([]string, error) {
	// TODO only returns values, but sometimes name is important too...
	//		like for eingangsdatum?

	documentTypex, err := qq.Data.QueryDocumentType().Only(ctx)
	if err != nil {
		if !enttenant.IsNotFound(err) {
			log.Println(err)
		}
		return nil, err
	}

	var filenameElems []string
	filenameElems = append(filenameElems, documentTypex.Name)

	// TODO ordner, user defined or at least date always first
	// TODO define date format?
	for _, assignment := range qq.identifyingPropertyAssignment(ctx) {
		propertym := mproperty.NewProperty(assignment.Edges.Property)
		assignmentm := mproperty.NewPropertyAssignment(assignment)

		if propertym.Data.Type == fieldtype.Checkbox && assignment.BoolValue == false {
			// don't add unselected checkboxes
			continue
		}

		filenameElems = append(filenameElems, assignmentm.String(ctx, propertym))
	}

	for _, tag := range qq.identifyingTagGroups(ctx) {
		filenameElems = append(filenameElems, tag.Name)
	}

	return filenameElems, nil
}

func (qq *File) identifyingPropertyAssignment(ctx ctxx.Context) []*enttenant.FilePropertyAssignment {
	// TODO check if it has a document type?

	// TODO does this work this way? and if so, is it efficient?
	identifiers := qq.Data.QueryDocumentType().
		QueryAttributes().
		Where(attribute.IsNameGiving(true)).
		QueryProperty().
		IDsX(ctx)
	assignments := qq.Data.QueryPropertyAssignment().
		WithProperty().
		Where(filepropertyassignment.PropertyIDIn(identifiers...)).
		AllX(ctx)

	// TODO ordner
	return assignments
}

func (qq *File) identifyingTagGroups(ctx ctxx.Context) []*enttenant.Tag {
	identifyingGroups := qq.Data.QueryDocumentType().
		QueryAttributes().
		Where(attribute.IsNameGiving(true)).
		QueryTag().
		IDsX(ctx)
	tags := qq.Data.QueryTags().
		Where(tag.GroupIDIn(identifyingGroups...)).
		AllX(ctx)

	return tags
}

func (qq *File) CurrentVersion(ctx context.Context) *StoredFile {
	// TODO is this okay or use a cache in ctx?
	if qq.nilableCurrentVersion != nil {
		return qq.nilableCurrentVersion
	}

	// TODO handle if File is Directory

	version := qq.Data.QueryVersions().Order(storedfile.ByCreatedAt(sql.OrderDesc())).Limit(1).OnlyX(ctx)
	qq.nilableCurrentVersion = NewStoredFile(version) // TODO inject factory into ctx?

	return qq.nilableCurrentVersion
}

// TODO is this okay?
func (qq *File) Versions(ctx ctxx.Context) []*StoredFile {
	versionsx := qq.Data.QueryVersions().AllX(ctx)
	var versions []*StoredFile

	for _, versionx := range versionsx {
		versions = append(versions, NewStoredFile(versionx))
	}

	return versions
}

func (qq *File) Parent(ctx ctxx.Context) (*File, error) {
	if qq.Data.ParentID == 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "file has no parent")
	}

	// TODO is this okay or use a cache in ctx?
	if qq.nilableParent != nil {
		return qq.nilableParent, nil
	}

	if qq.Data.Edges.Parent != nil {
		qq.nilableParent = NewFile(qq.Data.Edges.Parent)
		return qq.nilableParent, nil
	}

	// TODO does this set Edges?
	parentx := qq.Data.QueryParent().OnlyX(ctx)
	qq.nilableParent = NewFile(parentx)

	return qq.nilableParent, nil
}

func (qq *File) IsZIPArchive(ctx ctxx.Context) bool {
	if qq.Data.IsDirectory {
		return false
	}
	return qq.CurrentVersion(ctx).IsZIPArchive()
}

func (qq *File) IsMovedToFinalDestination(ctx ctxx.Context) bool {
	return qq.CurrentVersion(ctx).IsMovedToFinalDestination()
}
