# File Hashes

SimpleDMS stores two SHA-256 values for uploaded file content.

`stored_files.content_sha256` is the SHA-256 of the original uploaded bytes before storage transforms such as compression or encryption. It is the user-facing hash shown in the file info tab as `SHA-256 hash`, and it is the value used for exact duplicate detection.

`stored_files.sha256` is the checksum for the stored object representation. Depending on storage settings, this can describe compressed or encrypted bytes rather than the original file bytes, so it must not be used for user-facing file identity or duplicate detection.

When adding upload or import paths, compute and persist `content_sha256` while streaming the original bytes into storage. Existing objects without `content_sha256` are backfilled by the scheduler.
