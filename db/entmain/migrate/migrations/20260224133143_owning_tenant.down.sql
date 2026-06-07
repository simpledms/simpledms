-- reverse: create index "tenantaccountassignment_account_id_is_owning_tenant" to table: "tenant_account_assignments"
DROP INDEX `tenantaccountassignment_account_id_is_owning_tenant`;
-- reverse: create index "tenantaccountassignment_account_id_is_default" to table: "tenant_account_assignments"
DROP INDEX `tenantaccountassignment_account_id_is_default`;
-- reverse: create index "tenantaccountassignment_tenant_id_account_id" to table: "tenant_account_assignments"
DROP INDEX `tenantaccountassignment_tenant_id_account_id`;
-- reverse: create "new_tenant_account_assignments" table
DROP TABLE `new_tenant_account_assignments`;
