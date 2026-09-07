-- reverse: create index "documentnote_file_id" to table: "document_notes"
DROP INDEX `documentnote_file_id`;
-- reverse: create index "documentnote_space_id" to table: "document_notes"
DROP INDEX `documentnote_space_id`;
-- reverse: create index "document_notes_replaced_by_id_key" to table: "document_notes"
DROP INDEX `document_notes_replaced_by_id_key`;
-- reverse: create index "document_notes_public_id_key" to table: "document_notes"
DROP INDEX `document_notes_public_id_key`;
-- reverse: create "document_notes" table
DROP TABLE `document_notes`;
