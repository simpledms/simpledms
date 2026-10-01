-- reverse: add column "public_id" to table: "tags"
ALTER TABLE `tags` DROP COLUMN `public_id`;
-- reverse: add column "public_id" to table: "properties"
ALTER TABLE `properties` DROP COLUMN `public_id`;
-- reverse: add column "public_id" to table: "document_types"
ALTER TABLE `document_types` DROP COLUMN `public_id`;
