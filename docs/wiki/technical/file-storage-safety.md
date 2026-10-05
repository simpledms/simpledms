# File Storage Safety

SimpleDMS uses fail-safe retention for every upload, import, conversion, copy, and
cleanup path: when storage or transaction state is uncertain, keep the verified
readable bytes and retry reconciliation.

Temporary bytes are deleted only after a verified final copy is durably referenced.
Cleanup requires proof that an object is temporary, old, and unreferenced. Partial
cross-database conversions remain pinned to their destination tenant, and recovery
prefers committed file state over cleanup.

The authoritative developer rules are documented in
[File Storage Safety Invariants](../../docs/invariants/file_storage_safety.md).
