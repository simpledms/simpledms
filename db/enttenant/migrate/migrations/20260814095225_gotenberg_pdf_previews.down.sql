-- reverse: create index "previewconversion_status_next_attempt_at_id" to table: "preview_conversions"
DROP INDEX `previewconversion_status_next_attempt_at_id`;
-- reverse: create index "previewconversion_source_stored_file_id" to table: "preview_conversions"
DROP INDEX `previewconversion_source_stored_file_id`;
-- reverse: create "preview_conversions" table
DROP TABLE `preview_conversions`;
