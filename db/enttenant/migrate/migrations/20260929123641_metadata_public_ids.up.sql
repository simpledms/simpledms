-- add column "public_id" to table: "document_types"
ALTER TABLE `document_types` ADD COLUMN `public_id` text NULL;
-- add column "public_id" to table: "properties"
ALTER TABLE `properties` ADD COLUMN `public_id` text NULL;
-- add column "public_id" to table: "tags"
ALTER TABLE `tags` ADD COLUMN `public_id` text NULL;
