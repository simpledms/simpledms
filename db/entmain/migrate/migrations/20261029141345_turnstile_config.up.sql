-- add column "turnstile_site_key" to table: "system_configs"
ALTER TABLE `system_configs` ADD COLUMN `turnstile_site_key` text NULL;
-- add column "turnstile_secret" to table: "system_configs"
ALTER TABLE `system_configs` ADD COLUMN `turnstile_secret` blob NULL;
