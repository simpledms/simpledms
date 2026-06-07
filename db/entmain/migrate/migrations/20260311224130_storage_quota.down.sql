-- reverse: add column "max_upload_size_mib_override" to table: "tenants"
ALTER TABLE `tenants` DROP COLUMN `max_upload_size_mib_override`;
-- reverse: create "new_system_configs" table
DROP TABLE `new_system_configs`;
