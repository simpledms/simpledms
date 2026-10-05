# Recursive Browse Filter Performance

Recursive Browse filters currently avoid collecting every descendant file ID.
For non-root directories, the query builds a recursive CTE containing only directories in scope and then filters results with `files.parent_id IN (scope directories)`.

This improves search, tag, document type, and property filters when a folder contains many files but fewer folders.

Root directory filters use a separate shortcut: the space root already spans the whole space, so the query only excludes the root directory itself and avoids a recursive CTE.

## Drawbacks

- The non-root scope CTE is still linear in the number of folders below the selected directory.
- Broad filters can still be slow when a subtree contains many folders.
- The query still validates scope during each filtered request instead of using a persisted tree index.
- FTS and tag filters are not fully candidate-driven for non-root directories because scope validation still needs the folder CTE.

## Future Fix

Use a materialized closure table for file tree scope checks:

```sql
file_tree_closure (
  space_id INTEGER NOT NULL,
  ancestor_id INTEGER NOT NULL,
  descendant_id INTEGER NOT NULL,
  depth INTEGER NOT NULL
)
```

With indexes on `(space_id, ancestor_id, descendant_id)` and `(space_id, descendant_id, ancestor_id)`, recursive filters can validate scope with an indexed `EXISTS` lookup instead of rebuilding scope on every request.
