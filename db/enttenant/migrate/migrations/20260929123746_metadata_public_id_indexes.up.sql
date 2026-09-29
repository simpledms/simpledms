-- create index "document_types_public_id_key" to table: "document_types"
CREATE UNIQUE INDEX `document_types_public_id_key` ON `document_types` (`public_id`);
-- create index "properties_public_id_key" to table: "properties"
CREATE UNIQUE INDEX `properties_public_id_key` ON `properties` (`public_id`);
-- create index "tags_public_id_key" to table: "tags"
CREATE UNIQUE INDEX `tags_public_id_key` ON `tags` (`public_id`);
