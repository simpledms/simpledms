-- create "tenant_data_migrations" table
CREATE TABLE `tenant_data_migrations` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `key` text NOT NULL, `cursor` integer NOT NULL DEFAULT (0), `first_started_at` datetime NOT NULL, `completed_at` datetime NULL, `last_attempted_at` datetime NULL, `failed_at` datetime NULL, `last_error` text NULL, `retry_count` integer NOT NULL DEFAULT (0), `lease_token` text NULL, `lease_expires_at` datetime NULL);
-- create index "tenantdatamigration_key" to table: "tenant_data_migrations"
CREATE UNIQUE INDEX `tenantdatamigration_key` ON `tenant_data_migrations` (`key`);
