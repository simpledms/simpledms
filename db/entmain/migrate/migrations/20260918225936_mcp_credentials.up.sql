-- create "mcp_credentials" table
CREATE TABLE `mcp_credentials` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `public_id` text NOT NULL, `space_public_id` text NOT NULL, `label` text NOT NULL, `is_read_only` bool NOT NULL DEFAULT (true), `secret_hash` text NOT NULL, `created_at` datetime NOT NULL, `revoked_at` datetime NULL, `account_id` integer NOT NULL, `tenant_id` integer NOT NULL, CONSTRAINT `mcp_credentials_accounts_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON DELETE CASCADE, CONSTRAINT `mcp_credentials_tenants_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE);
-- create index "mcp_credentials_public_id_key" to table: "mcp_credentials"
CREATE UNIQUE INDEX `mcp_credentials_public_id_key` ON `mcp_credentials` (`public_id`);
-- create index "mcpcredential_account_id_created_at" to table: "mcp_credentials"
CREATE INDEX `mcpcredential_account_id_created_at` ON `mcp_credentials` (`account_id`, `created_at`);
