-- create index "attribute_space_id" to table: "attributes"
CREATE INDEX `attribute_space_id` ON `attributes` (`space_id`);
-- create index "documenttype_space_id" to table: "document_types"
CREATE INDEX `documenttype_space_id` ON `document_types` (`space_id`);
-- create index "filepropertyassignment_space_id" to table: "file_property_assignments"
CREATE INDEX `filepropertyassignment_space_id` ON `file_property_assignments` (`space_id`);
-- create index "property_space_id" to table: "properties"
CREATE INDEX `property_space_id` ON `properties` (`space_id`);
-- create index "spaceuserassignment_space_id" to table: "space_user_assignments"
CREATE INDEX `spaceuserassignment_space_id` ON `space_user_assignments` (`space_id`);
-- create index "tag_space_id" to table: "tags"
CREATE INDEX `tag_space_id` ON `tags` (`space_id`);
-- create index "tagassignment_space_id" to table: "tag_assignments"
CREATE INDEX `tagassignment_space_id` ON `tag_assignments` (`space_id`);
-- create index "file_space_id" to table: "files"
CREATE INDEX `file_space_id` ON `files` (`space_id`);
-- create index "file_parent_id" to table: "files"
CREATE INDEX `file_parent_id` ON `files` (`parent_id`);
-- create index "file_is_directory" to table: "files"
CREATE INDEX `file_is_directory` ON `files` (`is_directory`);
-- create index "file_is_in_inbox" to table: "files"
CREATE INDEX `file_is_in_inbox` ON `files` (`is_in_inbox`);
-- create index "file_space_id_parent_id" to table: "files"
CREATE INDEX `file_space_id_parent_id` ON `files` (`space_id`, `parent_id`);
-- create index "fileversion_stored_file_id" to table: "file_versions"
CREATE INDEX `fileversion_stored_file_id` ON `file_versions` (`stored_file_id`);
