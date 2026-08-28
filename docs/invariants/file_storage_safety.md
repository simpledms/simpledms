# File Storage Safety Invariants

## Scope

These invariants apply to every file ingestion and storage path, including browser
uploads, Inbox uploads, file versions, shared/PWA uploads, URL imports, archive
extraction, WebDAV, account-temporary conversion, background copy, and scheduler
cleanup.

## Preserve Canonical Bytes When State Is Uncertain

Rule: SimpleDMS retains the verified readable object whenever it cannot prove that
another verified readable copy is durably referenced by committed database state.

Network errors, ambiguous transaction outcomes, unavailable tenant databases,
conversion-token mismatches, and verification failures are retryable states. They
must not trigger deletion of the only verified bytes or release a conversion to a
different tenant.

Known corrupt or incomplete destinations may be removed only while the verified
source remains readable.

## Promotion Before Cleanup

Rule: A temporary object remains canonical until the final object is read back,
its size and SHA-256 match the temporary row, and the final-location database state
commits. Temporary-object deletion happens afterward and is idempotent.

## Recovery Prefers Success

Rule: If tenant file state commits before corresponding main-database state,
recovery completes or repairs the main marker. It never deletes the successful
tenant File, StoredFile, or object.

Conversion claims remain pinned to their destination tenant while successful state
may exist but cannot be inspected safely.

## Cleanup Requires Proof

Rule: Automated cleanup deletes only temporary-prefix objects that are old and
unreferenced by every active, successful, copy-pending, conversion, or preview
workflow. Final-prefix objects are never orphan-scanned.

Missing objects are an idempotent cleanup success. Any other storage error leaves
the database cleanup marker unchanged for retry.

## Explicit Deletion

Rule: An authorized explicit deletion removes the current object, retained object
versions, and delete markers. This is distinct from recovery cleanup and is the
only path that intentionally removes verified canonical bytes without promoting a
replacement.

## Verification

Minimum regression coverage includes authoritative stored-byte SHA-256, copy
fallback, retained-version deletion, missing-object cleanup, conversion claim
pinning, stale-worker protection, authorization rechecks, and concurrent uploads.

Detailed ingestion and recovery rules remain in
[WebDAV Inbox Ingestion Invariants](webdav_inbox_ingestion.md).
