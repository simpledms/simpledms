package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// TODO rename to FileInfo or FileLookup, info is maybe more general, than we can add some metadata
type FileInfo struct {
	ent.View
}

// manually executed in main.go
func (FileInfo) SQL() string {
	// language=sqlite
	return `
DROP VIEW IF EXISTS file_infos; CREATE VIEW file_infos AS WITH RECURSIVE file_infos(space_id, file_id, public_file_id, full_path, path, public_path) AS (
	-- base case
	SELECT 
		space_id,
		id, 
		public_id,
		'' AS full_path, 
		JSON_ARRAY(id) AS path,
		JSON_ARRAY(public_id) AS public_path
	FROM files WHERE is_directory AND parent_id IS NULL
		
	UNION ALL
	
	-- recursive case
	SELECT 
		f.space_id,
		f.id, 
		f.public_id,
		CASE fi.full_path WHEN '' THEN
			f.name
		ELSE
			fi.full_path || '/' || f.name
		END,
		JSON_INSERT(fi.path, '$[#]', f.id), /* $[#] means at the end */
		JSON_INSERT(fi.public_path, '$[#]', f.public_id)
	FROM files f
	JOIN file_infos fi ON f.parent_id = fi.file_id
)
SELECT * FROM file_infos
`

}

// TODO find a better name
// TODO had to execute manually, because it didn't work;
//
//	should be just the query without `create view...`
//
// TODO find a better name for path?
//
// TODO how to handle deleted_at? add column and mixing? filter out by default? latter probably not a good idea;
//
//	indirectly done by querying files table in all relevant cases?
//
// TODO only works with atlas
func (FileInfo) Annotations() []schema.Annotation {
	// CREATE VIEW file_paths AS
	// TODO use root file predicate for parent_id?
	return []schema.Annotation{}
}

func (FileInfo) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("space_id"),
		field.Int64("file_id"),
		field.String("public_file_id"), // TODO index or okay because files table is indexed?
		field.String("full_path"),
		field.JSON("path", []int64{}),
		field.JSON("public_path", []string{}),
	}
}

func (FileInfo) Mixin() []ent.Mixin {
	return []ent.Mixin{
		// NewSpaceMixin(),
	}
}

func (FileInfo) Policy() ent.Policy {
	// cannot use Mixin because edge seems not supported in views
	return NewSpaceMixin().Policy()
}

func (FileInfo) Edges() []ent.Edge {
	return []ent.Edge{
		// edge.To("file", File.Type).Field("file_id").Required().StorageKey(edge.Column("file_id")),
	}
}
