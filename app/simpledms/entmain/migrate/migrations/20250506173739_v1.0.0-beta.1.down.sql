-- reverse: create index "tenantaccountassignment_account_id_is_default" to table: "tenant_account_assignments"
DROP INDEX `tenantaccountassignment_account_id_is_default`;
-- reverse: create index "tenantaccountassignment_tenant_id_account_id" to table: "tenant_account_assignments"
DROP INDEX `tenantaccountassignment_tenant_id_account_id`;
-- reverse: create "tenant_account_assignments" table
DROP TABLE `tenant_account_assignments`;
-- reverse: create index "tenants_public_id_key" to table: "tenants"
DROP INDEX `tenants_public_id_key`;
-- reverse: create "tenants" table
DROP TABLE `tenants`;
-- reverse: create index "temporaryfile_upload_token" to table: "temporary_files"
DROP INDEX `temporaryfile_upload_token`;
-- reverse: create index "temporary_files_public_id_key" to table: "temporary_files"
DROP INDEX `temporary_files_public_id_key`;
-- reverse: create "temporary_files" table
DROP TABLE `temporary_files`;
-- reverse: create "system_configs" table
DROP TABLE `system_configs`;
-- reverse: create index "sessions_value_key" to table: "sessions"
DROP INDEX `sessions_value_key`;
-- reverse: create "sessions" table
DROP TABLE `sessions`;
-- reverse: create "mails" table
DROP TABLE `mails`;
-- reverse: create index "account_email" to table: "accounts"
DROP INDEX `account_email`;
-- reverse: create index "accounts_public_id_key" to table: "accounts"
DROP INDEX `accounts_public_id_key`;
-- reverse: create "accounts" table
DROP TABLE `accounts`;
