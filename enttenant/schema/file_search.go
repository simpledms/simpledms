package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type FileSearch struct {
	ent.View
}

func (FileSearch) SQL() string {
	// language=sqlite
	return `
-- FIXME should only be done once... no recreation on every start
DROP TABLE IF EXISTS file_searches;
DROP TRIGGER IF EXISTS files_ai;
DROP TRIGGER IF EXISTS files_ad;
DROP TRIGGER IF EXISTS files_au;
--CREATE VIRTUAL TABLE file_searches USING fts5(file_id UNINDEXED, filename, is_directory UNINDEXED);
--INSERT INTO file_searches SELECT id,name, is_directory FROM files;
CREATE VIRTUAL TABLE file_searches USING fts5(name, ocr_content, content='files', content_rowid='id');
INSERT INTO file_searches(file_searches) VALUES ('rebuild'); -- sync index

-- Triggers to keep the FTS index up to date.
CREATE TRIGGER files_ai AFTER INSERT ON files BEGIN
  INSERT INTO file_searches(rowid, name, ocr_content) VALUES (new.id, new.name, new.ocr_content);
END;
CREATE TRIGGER files_ad AFTER DELETE ON files BEGIN
  INSERT INTO file_searches(file_searches, rowid, name, ocr_content) VALUES('delete', old.id, old.name, old.ocr_content);
END;
CREATE TRIGGER files_au AFTER UPDATE ON files BEGIN
  INSERT INTO file_searches(file_searches, rowid, name, ocr_content) VALUES('delete', old.id, old.name, old.ocr_content);
  INSERT INTO file_searches(rowid, name, ocr_content) VALUES (new.id, new.name, new.ocr_content);
END;
`
}

func (FileSearch) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("space_id"),
		// hack: necessary for search queries // TODO use .Other?
		field.String("file_searches"), // table name
		field.Float("rank"),           // virtual column, always negative, the lower the better
		field.Int64("rowid"),
		field.String("name"), // filename
		field.String("ocr_content"),
		// field.Bool("is_directory"), // TODO remove?
	}
}

func (FileSearch) Policy() ent.Policy {
	// cannot use Mixin because edge seems not supported in views
	return NewSpaceMixin().Policy()
}
