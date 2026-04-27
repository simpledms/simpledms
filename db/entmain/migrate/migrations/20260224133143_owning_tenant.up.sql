-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_tenant_account_assignments" table
CREATE TABLE `new_tenant_account_assignments` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `is_contact_person` bool NOT NULL DEFAULT (false), `is_default` bool NOT NULL DEFAULT (false), `is_owning_tenant` bool NOT NULL DEFAULT (false), `role` text NOT NULL, `expires_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, `tenant_id` integer NOT NULL, `account_id` integer NOT NULL, CONSTRAINT `tenant_account_assignments_accounts_creator` FOREIGN KEY (`created_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenant_account_assignments_accounts_updater` FOREIGN KEY (`updated_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenant_account_assignments_tenants_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenant_account_assignments_accounts_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION);
-- copy rows from old table "tenant_account_assignments" to new temporary table "new_tenant_account_assignments"
INSERT INTO `new_tenant_account_assignments` (`id`, `created_at`, `updated_at`, `is_contact_person`, `is_default`, `role`, `expires_at`, `created_by`, `updated_by`, `tenant_id`, `account_id`) SELECT `id`, `created_at`, `updated_at`, `is_contact_person`, `is_default`, `role`, `expires_at`, `created_by`, `updated_by`, `tenant_id`, `account_id` FROM `tenant_account_assignments`;
-- drop "tenant_account_assignments" table after copying rows
DROP TABLE `tenant_account_assignments`;
-- rename temporary table "new_tenant_account_assignments" to "tenant_account_assignments"
ALTER TABLE `new_tenant_account_assignments` RENAME TO `tenant_account_assignments`;
-- create index "tenantaccountassignment_tenant_id_account_id" to table: "tenant_account_assignments"
CREATE UNIQUE INDEX `tenantaccountassignment_tenant_id_account_id` ON `tenant_account_assignments` (`tenant_id`, `account_id`);
-- create index "tenantaccountassignment_account_id_is_default" to table: "tenant_account_assignments"
CREATE UNIQUE INDEX `tenantaccountassignment_account_id_is_default` ON `tenant_account_assignments` (`account_id`, `is_default`) WHERE `is_default` = true;
-- create index "tenantaccountassignment_account_id_is_owning_tenant" to table: "tenant_account_assignments"
CREATE UNIQUE INDEX `tenantaccountassignment_account_id_is_owning_tenant` ON `tenant_account_assignments` (`account_id`, `is_owning_tenant`) WHERE `is_owning_tenant` = true;
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
