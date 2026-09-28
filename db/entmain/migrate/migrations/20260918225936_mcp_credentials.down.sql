-- reverse: create index "mcpcredential_account_id_created_at" to table: "mcp_credentials"
DROP INDEX `mcpcredential_account_id_created_at`;
-- reverse: create index "mcp_credentials_public_id_key" to table: "mcp_credentials"
DROP INDEX `mcp_credentials_public_id_key`;
-- reverse: create "mcp_credentials" table
DROP TABLE `mcp_credentials`;
