-- reverse: create "tag_sub_tags" table
DROP TABLE `tag_sub_tags`;
-- reverse: create "file_versions" table
DROP TABLE `file_versions`;
-- reverse: create index "users_email_key" to table: "users"
DROP INDEX `users_email_key`;
-- reverse: create index "users_account_id_key" to table: "users"
DROP INDEX `users_account_id_key`;
-- reverse: create index "users_public_id_key" to table: "users"
DROP INDEX `users_public_id_key`;
-- reverse: create "users" table
DROP TABLE `users`;
-- reverse: create index "tagassignment_file_id_tag_id" to table: "tag_assignments"
DROP INDEX `tagassignment_file_id_tag_id`;
-- reverse: create "tag_assignments" table
DROP TABLE `tag_assignments`;
-- reverse: create index "tag_space_id_name_group_id" to table: "tags"
DROP INDEX `tag_space_id_name_group_id`;
-- reverse: create index "tag_space_id_name" to table: "tags"
DROP INDEX `tag_space_id_name`;
-- reverse: create "tags" table
DROP TABLE `tags`;
-- reverse: create index "stored_files_public_id_key" to table: "stored_files"
DROP INDEX `stored_files_public_id_key`;
-- reverse: create "stored_files" table
DROP TABLE `stored_files`;
-- reverse: create index "spaceuserassignment_user_id_is_default" to table: "space_user_assignments"
DROP INDEX `spaceuserassignment_user_id_is_default`;
-- reverse: create index "spaceuserassignment_space_id_user_id" to table: "space_user_assignments"
DROP INDEX `spaceuserassignment_space_id_user_id`;
-- reverse: create "space_user_assignments" table
DROP TABLE `space_user_assignments`;
-- reverse: create index "space_name" to table: "spaces"
DROP INDEX `space_name`;
-- reverse: create index "spaces_public_id_key" to table: "spaces"
DROP INDEX `spaces_public_id_key`;
-- reverse: create "spaces" table
DROP TABLE `spaces`;
-- reverse: create index "property_space_id_name" to table: "properties"
DROP INDEX `property_space_id_name`;
-- reverse: create "properties" table
DROP TABLE `properties`;
-- reverse: create index "filepropertyassignment_file_id_property_id" to table: "file_property_assignments"
DROP INDEX `filepropertyassignment_file_id_property_id`;
-- reverse: create "file_property_assignments" table
DROP TABLE `file_property_assignments`;
-- reverse: create index "file_space_id_is_root_dir" to table: "files"
DROP INDEX `file_space_id_is_root_dir`;
-- reverse: create index "files_public_id_key" to table: "files"
DROP INDEX `files_public_id_key`;
-- reverse: create "files" table
DROP TABLE `files`;
-- reverse: create "document_types" table
DROP TABLE `document_types`;
-- reverse: create index "attribute_space_id_document_type_id_property_id" to table: "attributes"
DROP INDEX `attribute_space_id_document_type_id_property_id`;
-- reverse: create index "attribute_space_id_document_type_id_tag_id" to table: "attributes"
DROP INDEX `attribute_space_id_document_type_id_tag_id`;
-- reverse: create index "attribute_space_id_document_type_id_name" to table: "attributes"
DROP INDEX `attribute_space_id_document_type_id_name`;
-- reverse: create "attributes" table
DROP TABLE `attributes`;
