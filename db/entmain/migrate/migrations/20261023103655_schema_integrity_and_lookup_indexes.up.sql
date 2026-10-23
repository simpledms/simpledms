-- create index "session_account_id" to table: "sessions"
CREATE INDEX `session_account_id` ON `sessions` (`account_id`);
-- create index "session_deletable_at" to table: "sessions"
CREATE INDEX `session_deletable_at` ON `sessions` (`deletable_at`);
-- create index "tenantaccountassignment_account_id" to table: "tenant_account_assignments"
CREATE INDEX `tenantaccountassignment_account_id` ON `tenant_account_assignments` (`account_id`);
-- create index "webdavcredential_tenant_id_space_public_id_label" to table: "web_dav_credentials"
CREATE UNIQUE INDEX `webdavcredential_tenant_id_space_public_id_label` ON `web_dav_credentials` (`tenant_id`, `space_public_id`, `label`);
