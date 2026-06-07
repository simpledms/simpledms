# DDD Tactical Refactoring Findings

## Scope and method

This review focused on tactical DDD patterns in the current Go codebase:

- Aggregates and invariants
- Application services vs. HTTP/controller responsibilities
- Repository boundaries
- Value objects and domain policies
- Domain events for cross-aggregate side effects

I reviewed `action/`, `model/`, `common/file_repository.go`, and `ctxx/` with emphasis on high-change and high-complexity files.

Quick baseline signals from the current code:

- `action/` currently contains many persistence-level operations (approximately 89 `.Query(...)` call sites and 62 direct `ctx.*Tx` usages in action handlers/partials).
- Several action files are very large and mix HTTP, UI composition, domain decisions, and persistence logic:
  - `action/browse/list_filter_properties_partial.go`
  - `action/browse/list_dir_partial.go`
  - `action/dashboard/dashboard_cards_partial.go`
  - `action/common/move_file.go`

## Strengths already present

Before refactoring, keep and build on these existing strengths:

- Clear business areas split between `model/main/*` and `model/tenant/*`.
- Some rich domain behavior already exists (for example `model/main/account/account.go`, `model/tenant/space/space.go`).
- Useful value object already present (`model/main/uploadlimit/upload_limit.go`).
- A good policy-style service exists (`model/main/tenantaccess/tenant_access_service.go`).

## Prioritized refactorings

## 1) High: Move workflow logic out of action handlers into application services

### Problem

Many action handlers are currently transaction scripts with business rules and persistence mutations. This makes behavior hard to test and easy to duplicate.

### Evidence

- `action/browse/file_version_from_inbox_cmd.go:95`
- `action/browse/unzip_archive_cmd.go:72`
- `action/inbox/upload_file_cmd.go:66`
- `action/browse/upload_file_cmd.go:76`
- `action/dashboard/dashboard_cards_partial.go:74`

### DDD-oriented refactoring

- Introduce explicit application services (use-case services) in `model/...` or `app/...` packages.
- Keep action handlers thin: parse request -> call use case -> map result to UI events/snackbars.
- Return structured result objects from use cases (for example `MergeFromInboxResult`, `UploadResult`) instead of mutating HTTP behavior directly.

### First slice

Start with `file_version_from_inbox_cmd` because the file already contains `// TODO should be from model` and has a clear business operation boundary.

## 2) High: Create a stronger File aggregate boundary

### Problem

File-related invariants are spread across many action files. The same aggregate (`File`, `FileVersion`, `FilePropertyAssignment`, inbox flags, OCR metadata) is mutated from different places.

### Evidence

- `action/browse/set_file_property_cmd.go:75`
- `action/browse/add_file_property_value_cmd.go:79`
- `action/browse/remove_file_property_cmd.go:58`
- `action/inbox/assign_file_cmd.go:108`
- `action/inbox/move_file_cmd.go:47`
- `action/inbox/mark_as_done_cmd.go:61`
- `action/browse/file_version_from_inbox_cmd.go:150`

### DDD-oriented refactoring

- Treat `File` as aggregate root for file lifecycle operations.
- Move operations to domain/application methods such as:
  - `AssignProperty(...)`
  - `RemoveProperty(...)`
  - `MergeVersionFromInbox(...)`
  - `MarkDoneInInbox(...)`
  - `MoveToDirectory(...)`
- Enforce invariants in one place (same space, inbox-only operations, non-directory rules, deletion rules, etc.).

### First slice

Unify add/set/remove property commands behind one domain method and one repository save path.

## 3) High: Extract query logic from UI partials into query services/specifications

### Problem

Read-model query construction (including SQL predicates and ranking/filter rules) is embedded in UI partials, which couples rendering to query mechanics.

### Evidence

- `action/browse/list_dir_partial.go:413`
- `action/browse/list_dir_partial.go:905`
- `action/inbox/files_list_partial.go:253`
- `action/inbox/list_inbox_assignment_suggestions_partial.go:66`
- `ctxx/main_context.go:63` (cross-tenant query traversal in context object)

### DDD-oriented refactoring

- Introduce query objects/read services, for example:
  - `FileListQuery`
  - `InboxQuery`
  - `AssignmentSuggestionQuery`
- Model filters as criteria/specification objects instead of ad-hoc SQL fragments in views.
- Keep partials responsible only for view state mapping.

### First slice

Extract the `ListDirPartial.filesListItems` query building to a dedicated query service and keep the same output DTOs initially.

## 4) High: Remove HTTP error coupling from domain/model code

### Problem

Many model/domain methods return HTTP-specific errors (`e.NewHTTPErrorf(...)`). This couples domain logic to delivery concerns and blocks reuse (CLI/jobs/tests).

### Evidence

- `model/main/account/account.go:68`
- `model/tenant/space/space.go:82`
- `model/main/uploadlimit/upload_limit.go:20`
- `model/tenant/filesystem/file_system.go:46`

### DDD-oriented refactoring

- Introduce domain error types (`ErrUnauthorizedAction`, `ErrDuplicateName`, `ErrInvalidState`, etc.).
- Map domain errors to HTTP in action/router layers.
- Keep translation/UI text and status-code concerns out of core domain objects.

### First slice

Apply this to one bounded area first (for example `model/tenant/space`) and add an adapter mapper in action handlers.

## 5) Medium: Strengthen Tag aggregate invariants and consistency rules

### Problem

Tag grouping and assignment rules are partially enforced and partially bypassed via direct action queries.

### Evidence

- `model/tenant/tagging/tag_service.go:171`
- `action/tagging/move_tag_to_group_cmd.go:91`
- `action/tagging/create_and_assign_tag_cmd.go:90`

### DDD-oriented refactoring

- Create explicit aggregate operations for tag hierarchy changes.
- Validate group constraints in one place (group type, self-reference, cycles, cross-space consistency).
- Keep UI forms from querying/manipulating tag hierarchy rules directly.

### First slice

Harden `MoveToGroup` with full invariant checks and route all move operations through it.

## 6) Medium: Split Tenant domain behavior from infrastructure provisioning

### Problem

`Tenant` currently owns both business behavior and infrastructure provisioning/migration concerns, creating a very broad aggregate role.

### Evidence

- `model/main/tenant/tenant.go:56` (init/provisioning)
- `model/main/tenant/tenant.go:154` (migration execution)
- `model/main/tenant/tenant.go:286` (DB init and bootstrap)

### DDD-oriented refactoring

- Keep `Tenant` entity focused on domain state/rules.
- Move DB lifecycle/migration/provisioning into dedicated services, for example:
  - `TenantProvisioningService`
  - `TenantMigrationService`
  - `TenantDatabaseFactory`

### First slice

Extract `ExecuteDBMigrations` into a separate infrastructure service called from the current flow.

## 7) Medium: Introduce repository interfaces beyond `FileRepository`

### Problem

Only one explicit repository exists, and many areas rely directly on ent queries in actions and models.

### Evidence

- `common/file_repository.go:12`
- `common/file_repository.go:21` (panic usage)
- direct ent access spread across actions and model services.

### DDD-oriented refactoring

- Define repository interfaces per aggregate boundary (File, Tag, Space, Account/TenantMembership where needed).
- Use repositories to encapsulate loading graph details and prevent controller-level persistence leakage.
- Replace panic-based paths with explicit error returns.

### First slice

Introduce a `FileRepository` interface and one concrete ent implementation; update file commands to depend on the interface.

## 8) Medium: Use domain events for cross-aggregate side effects

### Problem

Cross-aggregate side effects (mail, session invalidation) are currently invoked inline in services, coupling flows and making retries/idempotency harder.

### Evidence

- `model/main/tenantuser/tenant_user_service.go:73`
- `model/main/signup/sign_up_service.go:85`
- `model/main/tenantmembership/tenant_account_lifecycle_service.go:55`

### DDD-oriented refactoring

- Emit domain events from use cases (`UserCreated`, `AccountRemovedFromTenant`, `SignUpCompleted`).
- Handle side effects in event handlers (mailer/session invalidation), optionally with an outbox pattern.

### First slice

Start with user creation mail delivery (`tenant_user_service`) and keep synchronous dispatch initially.

## 9) Medium: Centralize authorization policies

### Problem

Role checks are repeated across actions and widgets, increasing risk of drift between command and UI behavior.

### Evidence

- `action/managetenantusers/create_user_cmd.go:68`
- `action/managetenantusers/delete_user_cmd.go:47`
- `action/managespaceusers/assign_user_to_space_cmd.go:63`
- `action/managespaceusers/unassign_user_from_space_cmd.go:58`
- `action/managetenantusers/user_context_menu_widget.go:26`

### DDD-oriented refactoring

- Introduce policy services/value objects (for example `TenantPolicy`, `SpacePolicy`).
- Reuse the same policy checks from command handlers and UI composition.

### First slice

Extract owner checks into one `SpacePolicy.CanManageUsers(...)` and replace duplicated checks in manage-space-users actions.

## 10) Medium: Add aggregate-focused tests before deeper refactors

### Problem

Core behavioral areas have limited direct model-level tests, which increases refactor risk.

### Evidence

Only a small set of model tests currently exist (mainly account rate limiting/upload-limit/filesystem).

### DDD-oriented refactoring

- Add behavior tests around invariants for:
  - file lifecycle transitions
  - space user assignment/unassignment rules
  - tenant membership/account lifecycle

### First slice

Create tests for `FileVersionFromInbox` behavior before moving logic out of action handlers.

## Suggested implementation roadmap

### Phase 1 (low risk, high return)

- Extract one file workflow use case (`merge from inbox`).
- Introduce domain error mapping in one aggregate (`Space`).
- Add invariant tests for extracted behavior.

### Phase 2

- Extract file list/inbox queries into read services.
- Introduce criteria/value objects for property filters.
- Centralize space/tenant authorization policy checks.

### Phase 3

- Split tenant provisioning/migration from tenant entity.
- Add domain event dispatch for mail/session side effects.
- Broaden repository interfaces and remove panic paths.

## Notes

- Refactor incrementally (vertical slices), not as a big-bang rewrite.
- Keep current UI behavior/HTMX event contracts stable while moving business logic inward.
- Prefer introducing adapters around existing ent queries first, then tightening aggregate boundaries.

## Phase 1 concrete delivery plan (PR slices)

This section turns Phase 1 into executable PRs with exact files and interfaces.

### PR 1 - Characterization tests for merge-from-inbox

Goal: lock current behavior before moving logic.

Files to add:

- `server/file_version_from_inbox_cmd_test.go`

Files to reuse (helpers):

- `server/edit_space_cmd_test.go`
- `server/space_permissions_test.go`

Test cases to implement:

- `TestFileVersionFromInboxCmd_MergesVersionAndDeletesSource`
- `TestFileVersionFromInboxCmd_RejectsSameSourceAndTarget`
- `TestFileVersionFromInboxCmd_RejectsSourceOutsideInbox`
- `TestFileVersionFromInboxCmd_RejectsSourceWithoutVersion`

Acceptance criteria:

- No production code changes.
- Existing behavior is codified and green under `go test ./server -run FileVersionFromInbox`.

### PR 2 - Extract merge-from-inbox use case service

Goal: move business workflow out of `action/browse/file_version_from_inbox_cmd.go`.

Files to add:

- `model/tenant/file/file_version_from_inbox_service.go`
- `model/tenant/file/file_version_from_inbox_result.go`
- `model/tenant/file/file_domain_error.go`

Files to update:

- `action/browse/file_version_from_inbox_cmd.go`
- `action/browse/actions.go`

Interfaces and types:

```go
// model/tenant/file/file_version_from_inbox_service.go
type FileVersionFromInboxService interface {
	MergeFromInbox(ctx ctxx.Context, sourceFilePublicID string, targetFilePublicID string) (*FileVersionFromInboxResult, error)
}

type FileVersionFromInboxResult struct {
	TargetFileID   int64
	TargetPublicID string
	TargetName     string
	SourceFileID   int64
	SourcePublicID string
}
```

```go
// model/tenant/file/file_domain_error.go
var (
	ErrSourceFileRequired       = errors.New("source file is required")
	ErrTargetFileRequired       = errors.New("target file is required")
	ErrSourceAndTargetMustDiffer = errors.New("source and target must differ")
	ErrSourceMustBeInboxFile    = errors.New("source file must be in inbox")
	ErrFileHasNoVersions        = errors.New("file has no versions")
)
```

Implementation notes:

- Move method body currently in `mergeFromInbox(...)` from action to service.
- Keep action-level UX unchanged (same snackbar, same HX trigger values).
- Keep hard-delete behavior unchanged in this slice.

Acceptance criteria:

- PR 1 tests stay green.
- No API/route changes.

### PR 3 - Introduce domain errors for `Space` aggregate and map in actions

Goal: decouple `model/tenant/space` from HTTP error construction.

Files to add:

- `model/tenant/space/space_error.go`
- `action/managespaceusers/map_space_error.go`

Files to update:

- `model/tenant/space/space.go`
- `action/managespaceusers/assign_user_to_space_cmd.go`
- `action/managespaceusers/unassign_user_from_space_cmd.go`

Interfaces and types:

```go
// model/tenant/space/space_error.go
var (
	ErrUserAlreadyAssigned = errors.New("user already assigned to space")
	ErrCannotUnassignSelf  = errors.New("cannot unassign yourself from space")
)
```

```go
// action/managespaceusers/map_space_error.go
func mapSpaceError(err error) error
```

Implementation notes:

- `space.AssignUser(...)` and `space.UnassignUser(...)` return domain errors.
- Command handlers translate domain errors to `e.NewHTTPErrorf(...)`.
- Keep HTTP status/messages identical to current behavior.

Acceptance criteria:

- Existing manage-space-users flows continue working unchanged.
- New mapping has tests (see PR 4).

### PR 4 - Add invariant and permission tests for space user management

Goal: make refactor of `Space` safe and regression-proof.

Files to add:

- `server/space_user_assignment_test.go`

Test cases to implement:

- `TestAssignUserToSpaceCmd_RejectsDuplicateAssignment`
- `TestUnassignUserFromSpaceCmd_RejectsSelfUnassignment`
- `TestAssignUserToSpaceCmd_RequiresOwnerRole`

Acceptance criteria:

- `go test ./server -run SpaceUser` is green.
- Behavior before and after PR 3 is equivalent from HTTP perspective.

### Optional PR 5 - Lightweight repository seam for file use case

Goal: prepare for broader repository pattern adoption without changing behavior.

Files to add:

- `model/tenant/file/file_repository.go`

Files to update:

- `common/file_repository.go`
- `action/browse/file_version_from_inbox_cmd.go`

Interface:

```go
type FileRepository interface {
	GetX(ctx ctxx.Context, publicID string) *File
	GetWithParentX(ctx ctxx.Context, publicID string) *FileWithParent
}
```

Implementation notes:

- Keep methods compatible with current `common.FileRepository` implementation.
- Do not broaden scope to all actions yet.

## Suggested execution order

1. PR 1 (tests first).
2. PR 2 (extract merge use case, no behavior change).
3. PR 3 (space domain errors + action mapping).
4. PR 4 (space user assignment tests).
5. Optional PR 5 (repository seam).

## Validation commands per PR

- `go test ./server -run FileVersionFromInbox`
- `go test ./server -run SpaceUser`
- `go test ./server -run 'SpaceCreatePermissions|SpaceDeletePermissions'`
- `go test ./...`
