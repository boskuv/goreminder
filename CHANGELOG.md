# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

### Changed
- **POST /api/v1/tasks/{id}/done**: response body is now `TaskMarkedDoneResponse` (task DTO without `status`). Status is implied as `"done"`; omitting it avoids an extra repository fetch after the update.
 - Internal services now depend on a `queue.Publisher` interface instead of the concrete `queue.Producer`, allowing the message queue to be disabled while keeping DB logic intact.
 - Task queueing logic now uses typed `queue.TaskEvent`/`TaskEventType` contracts instead of ad-hoc maps, while preserving the existing Celery-compatible `{ "task": ..., "args": [...] }` payloads.
- **PUT /api/v1/tasks/{id}**: fixed recurrence update detection for partial updates. Editing non-schedule fields (for example `title`/`description`) no longer triggers implicit recurring start-date recalculation. `start_date` validation remains unchanged: when explicitly provided, it must be `now/future` (UTC).
- CRUD service logs for `task`, `backlog`, `target`, `user`, and `digest_settings` now include a unified audit format (`audit.operation`, `audit.entity`, `audit.entity_id`, `audit.actor_id`, `audit.changed_fields`, `audit.changed_count`) so updates can be understood without logging full request bodies.

### Added
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