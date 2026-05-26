# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

### Changed
- **Attachments service layout**: implementation moved out of this monorepo (`services/attachments` removed). gRPC contract and generated Go stubs live under `api/proto/attachments/v1` and `api/gen/attachments/v1`; core uses `pkg/attachments` as a gRPC client only. Regenerate stubs with `make proto-attachments` or `make proto-attachments-docker` (see [api/README.md](api/README.md)). Keep `attachments.proto` in sync with the attachments service repository.
- **DELETE /api/v1/tasks/{id}** and **DELETE /api/v1/users/{user_id}**: when `attachments.enabled` is true, core requests attachment purge after successful soft-delete (best-effort retry if the attachment service is temporarily unavailable).
- **POST /api/v1/tasks/{id}/done**: response body is now `TaskMarkedDoneResponse` (task DTO without `status`). Status is implied as `"done"`; omitting it avoids an extra repository fetch after the update.
 - Internal services now depend on a `queue.Publisher` interface instead of the concrete `queue.Producer`, allowing the message queue to be disabled while keeping DB logic intact.
 - Task queueing logic now uses typed `queue.TaskEvent`/`TaskEventType` contracts instead of ad-hoc maps, while preserving the existing Celery-compatible `{ "task": ..., "args": [...] }` payloads.
- **PUT /api/v1/tasks/{id}**: fixed recurrence update detection for partial updates. Editing non-schedule fields (for example `title`/`description`) no longer triggers implicit recurring start-date recalculation. `start_date` validation remains unchanged: when explicitly provided, it must be `now/future` (UTC).
- CRUD service logs for `task`, `backlog`, `target`, `user`, and `digest_settings` now include a unified audit format (`audit.operation`, `audit.entity`, `audit.entity_id`, `audit.actor_id`, `audit.changed_fields`, `audit.changed_count`) so updates can be understood without logging full request bodies.
- **PUT /api/v1/tasks/{id}** recurrence cleanup now supports recurring-to-single conversion in one update: when recurrence is removed from a recurring parent, only active child tasks are deleted (done/deleted children are preserved), and queue publish failures still rollback the transaction.
- **PUT /api/v1/tasks/{id}** now normalizes empty recurrence values (`cron_expression`, `rrule`) to `NULL` in DB instead of persisting empty strings.

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
  - **Core config** (`attachments.*`): `enabled`, `grpcAddr`, `timeout`, `directUploadMaxBytes`, `proxyDownloadEnabled`, `proxyDownloadMaxBytes`.
  - When `attachments.enabled` is `false`, all `/api/v1/tasks/{id}/attachments*` endpoints return **503** with `error: attachments_disabled`; task/user delete still succeeds (noop purge).
  - **Contract in repo**: `api/proto/attachments/v1/attachments.proto` + generated `api/gen/attachments/v1/*.pb.go` and `*_grpc.pb.go`.
- **Tasks — muting (`muted`)**: boolean column `tasks.muted` (migration `20260503160000_add_column_muted_to_tasks_table.sql`). **API**: optional `muted` on create/update (`POST /api/v1/tasks`, `PUT /api/v1/tasks/{id}`); dedicated `POST /api/v1/tasks/{id}/mute` and `POST /api/v1/tasks/{id}/unmute`. Task JSON responses (`TaskResponse`, `TaskMarkedDoneResponse`, tasks inside `GET /api/v1/digests`) always include the `muted` key (`true` / `false`). **Queue**: `TaskService.publishTaskEvent` skips publishing `worker.schedule_task` when the task row is muted; `worker.delete_task` is still published (mute/unmute and soft-delete flows rely on deletes reaching the worker). **Updates**: changing title, `start_date`, recurrence, etc. on a muted task updates the database but does not enqueue a new schedule while `muted` remains true; sending `muted: false` in the same `PUT` restores normal schedule publishing where applicable. **Autoreschedule**: daily job may still advance `start_date` in the DB for muted rows; schedule messages are not enqueued for them. Recurrence parents with `requires_confirmation` propagate mute/unmute to active children (see `MuteTask` / `UnmuteTask` in `task_service`).
- **Tasks — RRULE**: optional iCalendar **RRULE** string on tasks (`rrule` column, API field `rrule`) as an alternative to `cron_expression` for defining recurrence. `cron_expression` and `rrule` cannot both be set. RRULE strings are parsed and validated with [teambition/rrule-go](https://github.com/teambition/rrule-go); `DTSTART` defaults to the task’s `start_date` when omitted in the rule. Same parent/child model as cron: with `requires_confirmation`, the parent keeps the rule and children get the next concrete `start_date`; `RescheduleCronTasks` and “mark done → next child” use the next occurrence from either cron or RRULE. Database migration: `tasks.rrule` (TEXT).
- **GET /api/v1/backlogs**: добавлен query-параметр `completed` для фильтрации по статусу завершения (`completed=false` — только незавершённые, `completed=true` — только завершённые).
 - New `producer.enabled` configuration flag; when set to `false`, the app runs in DB-only mode using a no-op publisher instead of RabbitMQ.
 - Explicit queue contract type `queue.TaskMessage` for task-related messages, preserving the existing `{ "task": ..., "args": [...] }` JSON format used by workers.
 - Constants `ScheduledTaskActionSchedule` and `ScheduledTaskActionDelete` for `internal/models.ScheduledTask`, used by `TaskService.QueueTask` to dispatch scheduling vs deletion behavior via a `switch`.
- Internal shared audit helper in service layer for safe change summaries:
  - map-based diff (`old/new`) for update operations
  - field snapshots for create/delete operations
  - masking of sensitive user data in audit payloads (password hash value is never logged)

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
[v0.1.0]: https://github.com/boskuv/goreminder/releases/tag/v0.1.0