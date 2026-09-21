# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

### Added
- **Tasks — pre-remind (`pre_remind_before_seconds`)**: optional offset (seconds before `start_date`) for a preliminary reminder (migration `20260920220000_add_column_pre_remind_before_seconds_to_tasks_table.sql`). **API**: optional on create/update (`POST/PUT /api/v1/tasks`); `0` on update clears the option (max 30 days). Propagates to recurrence children. **Queue**: 9th `worker.schedule_task` arg (`null` when unset). **Worker** (`examples/worker`): schedules a second job `{messenger}_{task_id}_pre`; `delete_task` / mute removes both; pre webhook has no done/later buttons. Mute still suppresses all schedule publishes for the task.
- **Users — last activity**: column `users.last_activity_at` (migration `20260918190000_add_column_last_activity_at_to_users_table.sql`). Best-effort `ActivityTracker` updates it on successful user-driven mutations (tasks CRUD / mute / unmute / done / queue / attachments, user create/update, backlogs, targets, digest settings, messenger-related user create). Autoreschedule/scheduler and GET requests do not touch activity. **API**: `GET /api/v1/users/activity?limit=` (default/max 100) returns `[{user_id, name, last_activity_at}]`; optional `last_activity_at` on `UserResponse` when set.
- **Examples — sample worker**: `examples/worker` consumes Celery-style `{task, args}` from RabbitMQ (`worker.schedule_task` / `worker.delete_task`), stores due times in a Redis ZSET, and POSTs the production-compatible `/send_message` webhook payload.
- **Examples — telegram-bot webhook**: `examples/telegram-bot` listens on `POST /send_message` (default `:8001`) and handles `done:` / `later:` inline callbacks so the sample worker can deliver end-to-end.

## [v0.2.0] - 2026-09-15

### Added
- **Task attachments** (S3-compatible object storage; **attachment service** is a separate deployable, **core** is REST BFF):
  - **REST** (GoReminder API): `GET/POST /api/v1/tasks/{id}/attachments`, `POST .../attachments/{attachment_id}/complete`, `GET .../attachments/{attachment_id}/download`, `GET .../attachments/{attachment_id}/content`, `DELETE .../attachments/{attachment_id}`.
  - **Presigned upload**: `POST` with JSON (`original_name`, `content_type`, `size_bytes`) → `upload_url`, `status: pending` → client `PUT` to object storage → `POST .../complete` → `ready`.
  - **Hybrid direct upload**: same `POST /api/v1/tasks/{id}/attachments` with `multipart/form-data` (field `file`, optional `idempotency_key`; max size `attachments.directUploadMaxBytes`, default 2 MiB) → gRPC `UploadDirect` → `status: ready` without `complete`.
  - **Proxy download**: `GET .../attachments/{attachment_id}/content` streams file bytes through the API when `attachments.proxyDownloadEnabled` is true and size ≤ `attachments.proxyDownloadMaxBytes` (gRPC `DownloadDirect`); otherwise use presigned `.../download`.
  - **GET /api/v1/tasks/{id}**: `TaskDetailResponse` with optional `attachments` when `attachments.enabled` is true (field omitted when empty); list/create/update/mark-done responses use `TaskResponse` without attachments.
  - **Task history**: `attachment_added` when an attachment becomes `ready` (`UploadDirect`, `CompleteUpload`); `attachment_removed` on `DELETE .../attachments/{id}` (metadata in `old_value` / `new_value`; presigned `InitUpload` pending is not logged).
  - **gRPC client** (`pkg/attachments`): `InitUpload`, `UploadDirect`, `CompleteUpload`, `ListAttachments`, `GetDownloadURL`, `DownloadDirect`, `DeleteAttachment`, `PurgeByTask`, `PurgeByUser`; noop client when disabled.
  - **Purge on delete**: soft-delete task/user triggers `PurgeByTask` / `PurgeByUser` (best-effort); S3 object removal is asynchronous in the attachment service (transactional outbox).
  - **Core config** (`attachments.*`): `enabled`, `grpcAddr`, `timeout`, `directUploadMaxBytes`, `proxyDownloadEnabled`, `proxyDownloadMaxBytes`, `purgeOnTaskDone` (optional purge after `POST .../done`; default `false`).
  - When `attachments.enabled` is `false`, all `/api/v1/tasks/{id}/attachments*` endpoints return **503** with `error: attachments_disabled`; task/user delete still succeeds (noop purge).
  - **Contract in repo**: `api/proto/attachments/v1/attachments.proto` + generated `api/gen/attachments/v1/*.pb.go` and `*_grpc.pb.go`.
- **Tasks — muting (`muted`)**: boolean column `tasks.muted` (migration `20260503160000_add_column_muted_to_tasks_table.sql`). **API**: optional `muted` on create/update (`POST /api/v1/tasks`, `PUT /api/v1/tasks/{id}`); dedicated `POST /api/v1/tasks/{id}/mute` and `POST /api/v1/tasks/{id}/unmute`. Task JSON responses (`TaskResponse`, `TaskMarkedDoneResponse`, tasks inside `GET /api/v1/digests`) always include the `muted` key (`true` / `false`). **Queue**: `TaskService.publishTaskEvent` skips publishing `worker.schedule_task` when the task row is muted; `worker.delete_task` is still published. Recurrence parents with `requires_confirmation` propagate mute/unmute to active children.
- **Tasks — RRULE**: optional iCalendar **RRULE** string on tasks (`rrule` column, API field `rrule`) as an alternative to `cron_expression`. `cron_expression` and `rrule` cannot both be set. Parsed/validated with [teambition/rrule-go](https://github.com/teambition/rrule-go); same parent/child model as cron for confirmation flows, autoreschedule, and “mark done → next child”.
- **`producer.enabled`**: when `false`, the app runs in DB-only mode with a no-op publisher instead of RabbitMQ. Services depend on `queue.Publisher` / typed `queue.TaskEvent` contracts while keeping Celery-compatible `{ "task": ..., "args": [...] }` payloads.
- **GET /api/v1/backlogs**: query parameter `completed` (`true` / `false`) to filter by completion.
- **List/digest filters**: optional `messenger_user_id` on `GET /api/v1/users/{user_id}/tasks`, `GET /api/v1/backlogs`, `GET /api/v1/targets`, `GET /api/v1/digests`, and `GET /api/v1/digests/settings/all` (external ID resolved to internal MRU rows).
- **GET /api/v1/users/{user_id}/tasks**: optional `messenger_related_user_id`; when both this and `messenger_user_id` are set, `messenger_related_user_id` wins.
- **GET /api/v1/messengerRelatedUsers/all**: optional `user_id` and `chat_id` filters.
- Unified **audit logging** for CRUD on `task`, `backlog`, `target`, `user`, and `digest_settings` (`audit.operation`, `audit.entity`, `audit.entity_id`, `audit.actor_id`, `audit.changed_fields`, `audit.changed_count`), with password-hash masking in audit payloads.
- **CI** (GitHub Actions): `go test ./...` with coverage summary; **golangci-lint** gate (v2) after errcheck/staticcheck cleanup.
- **Unit tests**: service coverage ~75% (including UpdateTask/DeleteTask, mute/unmute, scheduler paths); handler httptest templates for tasks/users/backlog/target/messenger; sqlmock coverage for task/user/backlog repositories.
- **Examples**: refreshed `examples/python-client` against current `/api/v1` (204 DELETE handling, mute/unmute, targets, attachments, MRU/digest contract); new sample `examples/telegram-bot` using that client.

### Changed
- **Attachments service layout**: implementation moved out of this monorepo (`services/attachments` removed). gRPC contract and generated stubs live under `api/proto/attachments/v1` and `api/gen/attachments/v1`; core uses `pkg/attachments` as a gRPC client only. Regenerate with `make proto-attachments` / `make proto-attachments-docker` (see [api/README.md](api/README.md)).
- **DELETE /api/v1/tasks/{id}** and **DELETE /api/v1/users/{user_id}**: when `attachments.enabled` is true, core requests attachment purge after successful soft-delete (best-effort).
- **POST /api/v1/tasks/{id}/done**: response body is `TaskMarkedDoneResponse` (task DTO without `status`; status implied as `"done"`).
- **PUT /api/v1/tasks/{id}**: recurrence update detection for partial updates — editing non-schedule fields no longer triggers implicit recurring start-date recalculation; empty `cron_expression` / `rrule` normalize to `NULL`; recurring-to-single conversion deletes only active children (done/deleted preserved), with queue publish failures rolling back the transaction.
- **README**: accuracy pass, CI/service-coverage badges, collapsible long sections; project tree documents `examples/`.
- **golangci-lint v2**: exclusions moved to `linters.exclusions` (`paths` / `rules`); config verifies under CI lint action v2.8.0.

### Fixed
- **PUT /api/v1/tasks/{id}**: metadata edits on recurring tasks (`cron_expression` or `rrule`, `requires_confirmation=false`) with an overdue `start_date` republish `worker.schedule_task` with the next occurrence so the worker stays in sync; one-time tasks with a past `start_date` still skip queue publish on metadata-only updates.
- **Task unmute** (`POST /api/v1/tasks/{id}/unmute`, `PUT` with `muted: false`): overdue recurring `start_date` is advanced to the next occurrence before republish; confirmation parents are not scheduled on unmute (active children are); `PUT` unmute matches `POST .../unmute` rules.
- **Autoreschedule**: muted recurring child tasks advance `start_date` to the parent’s next cron/RRULE occurrence (instead of `+24h`).
- Recurring schedule publishing / unmute past-date edge cases aligned with autoreschedule calculation.
- Optional purge of attachments on task done when `attachments.purgeOnTaskDone` is enabled.

### Dependencies
- Bump Go toolchain in Dockerfile; `pgx/v5`, `grpc`, OpenTelemetry, and related deps.

## [v0.1.0] - 2026-01-25
### Changed
- **Breaking**: Version management system now uses build-time injection
  - Version is now defined in VERSION file instead of hardcoded
  - Build process requires ldflags for version injection
  - Swagger documentation now uses dynamic versioning
- Refactored version handling into dedicated package (`pkg/version`)

### Added
- Makefile targets for version management (`make bump-version`, `make show-version`)
- Enhanced `/version` endpoint with build metadata

<!-- links -->
[Unreleased]: https://github.com/boskuv/goreminder/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/boskuv/goreminder/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/boskuv/goreminder/releases/tag/v0.1.0
