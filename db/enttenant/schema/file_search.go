package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type FileSearch struct {
	ent.View
}

// SQL rebuilds the search index, including rows written while triggers were missing.
func (qq FileSearch) SQL() string {
	query := `DROP TRIGGER IF EXISTS files_ai;
DROP TRIGGER IF EXISTS files_ad;
DROP TRIGGER IF EXISTS files_au;
DROP TABLE IF EXISTS file_searches;
` + qq.TableSQL() + "\nINSERT INTO file_searches(file_searches) VALUES ('rebuild');\n"
	triggers := qq.TriggersSQL()
	for _, name := range []string{"files_ai", "files_ad", "files_au"} {
		query += triggers[name] + "\n"
	}
	return query
}

// TableSQL defines the external-content FTS table without rebuilding its index.
func (FileSearch) TableSQL() string {
	// language=sqlite
	return `
CREATE VIRTUAL TABLE file_searches USING fts5(
  space_id UNINDEXED,
  is_directory UNINDEXED,
  is_in_inbox UNINDEXED,
  name,
  ocr_content,
  content='files',
  content_rowid='id',
  prefix='2 3 4',
  tokenize='porter unicode61'
);`
}

// TriggersSQL defines the triggers that keep indexed text synchronized with files.
func (FileSearch) TriggersSQL() map[string]string {
	return map[string]string{
		"files_ai": `CREATE TRIGGER files_ai AFTER INSERT ON files BEGIN
  INSERT INTO file_searches(rowid, space_id, is_directory, is_in_inbox, name, ocr_content)
  VALUES (new.id, new.space_id, new.is_directory, new.is_in_inbox, new.name, new.ocr_content);
END;`,
		"files_ad": `CREATE TRIGGER files_ad AFTER DELETE ON files BEGIN
  INSERT INTO file_searches(file_searches, rowid, space_id, is_directory, is_in_inbox, name, ocr_content)
  VALUES('delete', old.id, old.space_id, old.is_directory, old.is_in_inbox, old.name, old.ocr_content);
END;`,
		// Unindexed values are read from the external content table. Only changes to
		// indexed text or its rowid require retokenizing potentially large OCR content.
		"files_au": `CREATE TRIGGER files_au AFTER UPDATE OF id, name, ocr_content ON files
WHEN old.id IS NOT new.id OR old.name IS NOT new.name OR old.ocr_content IS NOT new.ocr_content
BEGIN
  INSERT INTO file_searches(file_searches, rowid, space_id, is_directory, is_in_inbox, name, ocr_content)
  VALUES('delete', old.id, old.space_id, old.is_directory, old.is_in_inbox, old.name, old.ocr_content);
  INSERT INTO file_searches(rowid, space_id, is_directory, is_in_inbox, name, ocr_content)
  VALUES (new.id, new.space_id, new.is_directory, new.is_in_inbox, new.name, new.ocr_content);
END;`,
	}
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
