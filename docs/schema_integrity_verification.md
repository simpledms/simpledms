# Schema integrity verification — 2026-10-23

## Generation

- `GOTOOLCHAIN=go1.26.4 go generate .` completed successfully.
- `GOTOOLCHAIN=go1.26.4 go run ./cmd/migrate schema_integrity_and_lookup_indexes`
  generated the main and tenant upgrades named
  `20261023103655_schema_integrity_and_lookup_indexes`.
- The tenant upgrade rebuilds `attributes` and `file_property_assignments` to add
  CHECK constraints; the other changes are index operations. Main migration generation
  also reconciled the existing WebDAV label-uniqueness definition with migration history.
- Downgrades were generated using the pinned Atlas library's reverse schema diff,
  rather than its incomplete per-statement reversal of SQLite table replacement.
  Checksums were regenerated through Atlas. No migration SQL was edited by hand.
- The installed Go 1.27.2 compiler's export format was incompatible with Ent's
  importer. Generation and tests use the repository's Go 1.26.4 version.

## Query-plan evidence

The scratch scripts `/tmp/opencode/index-check.py` and
`/tmp/opencode/index-check-followup.py` replayed the previous tenant migrations,
seeded 4,000 files across Spaces/folders/Inbox/deletion states, ran `ANALYZE`, and
compared `EXPLAIN QUERY PLAN` before and after index removal.

The tested browse name/newest/oldest and Inbox name/date queries, including bound
and literal booleans, retained their full `_fast` indexes without an additional
sort after removing `file_browse_name`, `file_browse_created`,
`file_browse_created_oldest`, `file_inbox_created`, and `file_inbox_name`.
Inbox ascending-date ordering already required a final tie-breaker sort.
This is synthetic query-plan evidence, not a production latency benchmark.
The separate parent and boolean-only indexes were retained.

## Tests

- The initial eight focused lifecycle, constraint, development-migration, and FTS
  regression tests passed across `db/entx`, `model/main/tenant`, and `server`.
- Upgrade/downgrade/re-upgrade data-preservation and invalid-legacy-data tests passed:
  `GOTOOLCHAIN=go1.26.4 go test ./db/entx -run '^(TestSchemaIntegrityMigrationsPreserveExistingData|TestSchemaIntegrityMigrationRejectsInvalidLegacyDataWithoutDeletingIt)$' -count=1`.
- Required submodule suite, including the legacy FTS-trigger upgrade: passed;
  all reported packages passed or had no test files.
  Command: `GOTOOLCHAIN=go1.26.4 go test ./... -skip '^(TestConcurrentUploadFileCmdWithSlowS3|TestUploadFileCmdFailsWhenS3Unavailable|TestUploadFilesCmdFailsWhenS3Unavailable)$'`.
  The three unrelated network-fault simulations are excluded as required by `AGENTS.md`.
  Output: `/tmp/opencode/simpledms-go-test.log`. No other skips were reported in that output.

## Migration boundaries

Existing metadata must satisfy the new CHECK constraints; migration failures retain
the original rows rather than inventing missing targets or discarding values.
Downgrading to globally unique user emails requires that email reuse has not introduced
duplicates across active and deleted users.

Search-index reconciliation is described in
[the FTS migration ADR](adrs/2026.10-preserve_fts_indexes_during_migrations.md).
