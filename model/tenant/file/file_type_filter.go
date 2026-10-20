package file

import (
	"log"
	"path"
	"slices"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entquery"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
)

// FileTypeWithoutExtension identifies files whose names have no extension.
const FileTypeWithoutExtension = "none"

// FileTypeFilter discovers and filters filename extensions in a Space.
type FileTypeFilter struct{}

// NewFileTypeFilter creates a filename-extension filter.
func NewFileTypeFilter() *FileTypeFilter {
	return &FileTypeFilter{}
}

// Available returns the distinct extensions of live files throughout the current Space.
func (qq *FileTypeFilter) Available(ctx ctxx.Context) ([]string, error) {
	// Read only names: type discovery must not load document content or file versions.
	names, err := ctx.TenantCtx().TTx.File.Query().
		Where(dbfile.SpaceID(ctx.SpaceCtx().Space.ID), entquery.FileIsDirectory(false)).
		Select(dbfile.FieldName).
		Strings(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	types := make(map[string]struct{})
	for _, name := range names {
		extension := strings.ToLower(path.Ext(name))
		if extension == "" {
			extension = FileTypeWithoutExtension
		}
		types[extension] = struct{}{}
	}
	available := make([]string, 0, len(types))
	for extension := range types {
		available = append(available, extension)
	}
	slices.Sort(available)
	return available, nil
}

// Apply adds the selected filename extensions to an existing file query.
func (qq *FileTypeFilter) Apply(
	query *enttenant.FileQuery,
	extensions []string,
) *enttenant.FileQuery {
	if len(extensions) == 0 {
		return query
	}
	predicates := make([]predicate.File, 0, len(extensions))
	for _, extension := range extensions {
		if extension == FileTypeWithoutExtension {
			predicates = append(predicates, dbfile.Not(dbfile.NameContains(".")))
			continue
		}
		predicates = append(predicates, func(selector *sql.Selector) {
			selector.Where(sql.HasSuffixFold(selector.C(dbfile.FieldName), extension))
		})
	}
	return query.Where(entquery.FileIsDirectory(false), dbfile.Or(predicates...))
}
