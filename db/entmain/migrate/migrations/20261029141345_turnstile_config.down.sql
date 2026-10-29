-- reverse: add column "turnstile_secret" to table: "system_configs"
ALTER TABLE `system_configs` DROP COLUMN `turnstile_secret`;
-- reverse: add column "turnstile_site_key" to table: "system_configs"
ALTER TABLE `system_configs` DROP COLUMN `turnstile_site_key`;
