-- reverse: create index "fileversion_stored_file_id" to table: "file_versions"
DROP INDEX `fileversion_stored_file_id`;
-- reverse: create index "file_space_id_parent_id" to table: "files"
DROP INDEX `file_space_id_parent_id`;
-- reverse: create index "file_is_in_inbox" to table: "files"
DROP INDEX `file_is_in_inbox`;
-- reverse: create index "file_is_directory" to table: "files"
DROP INDEX `file_is_directory`;
-- reverse: create index "file_parent_id" to table: "files"
DROP INDEX `file_parent_id`;
-- reverse: create index "file_space_id" to table: "files"
DROP INDEX `file_space_id`;
-- reverse: create index "tagassignment_space_id" to table: "tag_assignments"
DROP INDEX `tagassignment_space_id`;
-- reverse: create index "tag_space_id" to table: "tags"
DROP INDEX `tag_space_id`;
-- reverse: create index "spaceuserassignment_space_id" to table: "space_user_assignments"
DROP INDEX `spaceuserassignment_space_id`;
-- reverse: create index "property_space_id" to table: "properties"
DROP INDEX `property_space_id`;
-- reverse: create index "filepropertyassignment_space_id" to table: "file_property_assignments"
DROP INDEX `filepropertyassignment_space_id`;
-- reverse: create index "documenttype_space_id" to table: "document_types"
DROP INDEX `documenttype_space_id`;
-- reverse: create index "attribute_space_id" to table: "attributes"
DROP INDEX `attribute_space_id`;
